package sqlite

import (
	"errors"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	tencentkline "github.com/easyspace-ai/stock_api/pkg/tencentkline"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"

	"github.com/easyspace-ai/yilimi/internal/workbench/domain/stock"
	"github.com/easyspace-ai/yilimi/internal/workbench/infrastructure/cache"
	"github.com/easyspace-ai/yilimi/internal/workbench/infrastructure/clients"
	"github.com/easyspace-ai/yilimi/internal/workbench/infrastructure/clients/eastmoney"
	"github.com/easyspace-ai/yilimi/internal/workbench/infrastructure/clients/sina"
	"github.com/easyspace-ai/yilimi/internal/workbench/ports"
)

// StockRepositoryImpl 股票仓储实现
type StockRepositoryImpl struct {
	db         *gorm.DB
	cache      *cache.MultiLayerCache
	sinaClient *sina.Client
	emClient   *eastmoney.Client
	config     *clients.SettingConfig
}

// NewStockRepository 创建股票仓储实例
func NewStockRepository(
	db *gorm.DB,
	cache *cache.MultiLayerCache,
	config *clients.SettingConfig,
) ports.StockRepository {
	return &StockRepositoryImpl{
		db:         db,
		cache:      cache,
		sinaClient: sina.NewClient(),
		emClient:   eastmoney.NewClient(config),
		config:     config,
	}
}

// GetByCode 根据代码获取股票信息
func (r *StockRepositoryImpl) GetByCode(code string) (*stock.StockInfo, error) {
	cacheKey := cache.GenerateKey("stock", "info", code)
	var result stock.StockInfo

	if r.cache.Get(cacheKey, &result) {
		return &result, nil
	}

	var info stock.StockInfo
	err := r.db.Where("code = ?", code).First(&info).Error
	if err != nil && err != gorm.ErrRecordNotFound {
		return nil, err
	}

	if err == nil {
		r.cache.Set(cacheKey, &info, 1*time.Hour)
		return &info, nil
	}

	return nil, fmt.Errorf("stock not found: %s", code)
}

// GetByCodes 批量获取股票信息
func (r *StockRepositoryImpl) GetByCodes(codes []string) ([]stock.StockInfo, error) {
	var results []stock.StockInfo
	err := r.db.Where("code IN ?", codes).Find(&results).Error
	return results, err
}

// List 获取股票列表
func (r *StockRepositoryImpl) List(market, industry, concept string, page, pageSize int) ([]stock.StockInfo, int64, error) {
	cacheKey := cache.GenerateKey("stock", "list", market, industry, concept, page, pageSize)
	var cached struct {
		List  []stock.StockInfo
		Total int64
	}

	if r.cache.Get(cacheKey, &cached) {
		return cached.List, cached.Total, nil
	}

	query := r.db.Model(&stock.StockInfo{})
	if market != "" {
		query = query.Where("market = ?", market)
	}
	if industry != "" {
		query = query.Where("industry LIKE ?", "%"+industry+"%")
	}
	if concept != "" {
		query = query.Where("concept LIKE ?", "%"+concept+"%")
	}

	var total int64
	query.Count(&total)

	offset := (page - 1) * pageSize
	var results []stock.StockInfo
	err := query.Offset(offset).Limit(pageSize).Find(&results).Error

	if err == nil {
		r.cache.Set(cacheKey, struct {
			List  []stock.StockInfo
			Total int64
		}{results, total}, 5*time.Minute)
	}

	return results, total, err
}

// Search 搜索股票
func (r *StockRepositoryImpl) Search(keyword string) ([]stock.StockInfo, error) {
	// v2：先前会把空结果写入缓存，导致长期得不到数据；换 key 使旧缓存失效
	cacheKey := cache.GenerateKey("stock", "search", "v2", keyword)
	var results []stock.StockInfo

	if r.cache.Get(cacheKey, &results) {
		return results, nil
	}

	err := r.db.Where("code LIKE ? OR name LIKE ?", "%"+keyword+"%", "%"+keyword+"%").
		Limit(50).
		Find(&results).Error

	// 空结果不写入缓存，否则在库尚未同步时会把「无数据」固定 5 分钟
	if err == nil && len(results) > 0 {
		r.cache.Set(cacheKey, results, 5*time.Minute)
	}

	return results, err
}

// GetQuote 获取实时行情
func (r *StockRepositoryImpl) GetQuote(code string) (*stock.StockQuote, error) {
	quotes, err := r.GetQuotes([]string{code})
	if err != nil {
		return nil, err
	}
	if len(quotes) == 0 {
		return nil, fmt.Errorf("no quote data: %s", code)
	}
	return &quotes[0], nil
}

// GetQuotes 批量获取实时行情
func (r *StockRepositoryImpl) GetQuotes(codes []string) ([]stock.StockQuote, error) {
	cacheKey := cache.GenerateKey("stock", "quotes", strings.Join(codes, ","))
	var results []stock.StockQuote

	if r.cache.Get(cacheKey, &results) {
		return results, nil
	}

	sinaQuotes, err := r.sinaClient.GetQuotes(codes)
	if err != nil {
		return nil, err
	}

	for _, q := range sinaQuotes {
		quote := stock.StockQuote{
			Symbol:     q.Symbol,
			Name:       q.Name,
			Price:      q.Price,
			Change:     q.Price - q.PrevClose,
			ChangePct:  0,
			Open:       q.Open,
			High:       q.High,
			Low:        q.Low,
			PrevClose:  q.PrevClose,
			Volume:     q.Volume,
			Amount:     q.Amount,
			UpdateTime: q.Date + " " + q.Time,
		}
		if q.PrevClose > 0 {
			quote.ChangePct = (q.Price - q.PrevClose) / q.PrevClose * 100
		}
		results = append(results, quote)
	}

	r.cache.Set(cacheKey, results, 3*time.Second)
	return results, nil
}

// GetKLine 获取 K线数据
func (r *StockRepositoryImpl) GetKLine(code string, klineType string, days int, adjustFlag string) (*stock.KLineData, error) {
	cacheKey := cache.GenerateKey("stock", "kline", code, klineType, days, adjustFlag)
	var result stock.KLineData

	if r.cache.Get(cacheKey, &result) {
		return &result, nil
	}

	emType := eastmoney.KLineType(klineType)
	if emType == "" {
		emType = eastmoney.KLineTypeDay
	}

	symbol := normalizeDailyKlineSymbol(code)
	normalizedType := normalizeDailyKlineType(klineType)
	if normalizedType == "day" {
		cachedData, err := r.loadDailyKlineFromDB(symbol, normalizedType, adjustFlag, days)
		if err == nil && len(cachedData.List) >= days && days > 0 {
			r.cache.Set(cacheKey, *cachedData, 24*time.Hour)
			return cachedData, nil
		}
	}

	klineData, name, err := r.emClient.GetKLine(code, emType, adjustFlag, days)
	if err != nil {
		return nil, err
	}

	result = stock.KLineData{
		Code: code,
		Name: name,
	}

	for _, k := range klineData {
		result.List = append(result.List, stock.KLineItem{
			Date:   k.Date,
			Open:   k.Open,
			Close:  k.Close,
			High:   k.High,
			Low:    k.Low,
			Volume: k.Volume,
			Amount: k.Amount,
			Change: k.Change,
		})
	}

	if normalizedType == "day" && len(result.List) > 0 {
		_ = r.upsertDailyKlineToDB(symbol, name, normalizedType, adjustFlag, result.List)
	}

	r.cache.Set(cacheKey, result, 24*time.Hour)
	return &result, nil
}

func normalizeDailyKlineSymbol(code string) string {
	trimmed := strings.ToUpper(strings.TrimSpace(code))
	if len(trimmed) == 6 && !strings.Contains(trimmed, ".") {
		if strings.HasPrefix(trimmed, "6") || strings.HasPrefix(trimmed, "5") || strings.HasPrefix(trimmed, "9") {
			return trimmed + ".SH"
		}
		return trimmed + ".SZ"
	}
	if strings.HasPrefix(trimmed, "SH") || strings.HasPrefix(trimmed, "SZ") || strings.HasPrefix(trimmed, "BJ") {
		prefix := trimmed[:2]
		suffix := strings.TrimSpace(trimmed[2:])
		if suffix != "" {
			return suffix + "." + strings.ToUpper(prefix)
		}
	}
	return trimmed
}

func normalizeDailyKlineType(value string) string {
	switch strings.ToLower(strings.TrimSpace(value)) {
	case "day", "d", "101", "daily", "":
		return "day"
	case "week", "w", "102", "weekly":
		return "week"
	case "month", "m", "103", "monthly":
		return "month"
	default:
		return "day"
	}
}

func (r *StockRepositoryImpl) loadDailyKlineFromDB(symbol string, klineType string, adjustFlag string, days int) (*stock.KLineData, error) {
	var rows []stock.DailyKLineCache
	query := r.db.
		Model(&stock.DailyKLineCache{}).
		Where("symbol = ? AND k_line_type = ? AND adjust_flag = ?", symbol, klineType, adjustFlag).
		Order("date DESC")
	if days > 0 {
		query = query.Limit(days)
	}
	if err := query.Find(&rows).Error; err != nil {
		return nil, err
	}
	if len(rows) == 0 {
		return &stock.KLineData{
			Code: symbol,
			Name: symbol,
			List: []stock.KLineItem{},
		}, nil
	}
	list := make([]stock.KLineItem, 0, len(rows))
	for i := len(rows) - 1; i >= 0; i-- {
		row := rows[i]
		list = append(list, stock.KLineItem{
			Date:   row.Date,
			Open:   row.Open,
			Close:  row.Close,
			High:   row.High,
			Low:    row.Low,
			Volume: row.Volume,
			Amount: row.Amount,
			Change: row.Change,
		})
	}
	return &stock.KLineData{
		Code: symbol,
		Name: rows[0].Name,
		List: list,
	}, nil
}

func (r *StockRepositoryImpl) upsertDailyKlineToDB(symbol string, name string, klineType string, adjustFlag string, list []stock.KLineItem) error {
	items := make([]stock.DailyKLineCache, 0, len(list))
	for _, item := range list {
		items = append(items, stock.DailyKLineCache{
			Symbol:     symbol,
			Date:       item.Date,
			KLineType:  klineType,
			AdjustFlag: adjustFlag,
			Name:       name,
			Open:       item.Open,
			Close:      item.Close,
			High:       item.High,
			Low:        item.Low,
			Volume:     item.Volume,
			Amount:     item.Amount,
			Change:     item.Change,
		})
	}
	return r.db.Clauses(clause.OnConflict{
		Columns: []clause.Column{
			{Name: "symbol"},
			{Name: "date"},
			{Name: "k_line_type"},
			{Name: "adjust_flag"},
		},
		DoUpdates: clause.AssignmentColumns([]string{
			"name", "open", "close", "high", "low", "volume", "amount", "change", "updated_at",
		}),
	}).Create(&items).Error
}

// GetCommonKLine 获取兼容港股/美股/指数的通用K线
func (r *StockRepositoryImpl) GetCommonKLine(code string, klineType string, days int) (*stock.KLineData, error) {
	cacheKey := cache.GenerateKey("stock", "common-kline", code, klineType, days)
	var cached stock.KLineData
	if r.cache.Get(cacheKey, &cached) {
		return &cached, nil
	}

	raw, err := tencentkline.FetchCommonKLine(tencentkline.Config{
		Timeout: time.Duration(r.config.CrawlTimeOut) * time.Second,
	}, code, klineType, days)
	if err != nil {
		return nil, err
	}
	out := &stock.KLineData{
		Code: raw.Code,
		Name: raw.Name,
		List: make([]stock.KLineItem, len(raw.List)),
	}
	for i, it := range raw.List {
		out.List[i] = stock.KLineItem{
			Date: it.Date, Open: it.Open, Close: it.Close, High: it.High, Low: it.Low,
			Volume: it.Volume, Amount: it.Amount, Change: it.Change,
		}
	}
	r.cache.Set(cacheKey, *out, 24*time.Hour)
	return out, nil
}

// GetMinutePrice 获取分时数据
func (r *StockRepositoryImpl) GetMinutePrice(code string) ([]stock.KLineItem, error) {
	cacheKey := cache.GenerateKey("stock", "minute", code)
	var results []stock.KLineItem

	if r.cache.Get(cacheKey, &results) {
		return results, nil
	}

	klineData, _, err := r.emClient.GetKLine(code, eastmoney.KLineType1Min, "", 240)
	if err != nil {
		return nil, err
	}

	for _, k := range klineData {
		results = append(results, stock.KLineItem{
			Date:   k.Date,
			Open:   k.Open,
			Close:  k.Close,
			High:   k.High,
			Low:    k.Low,
			Volume: k.Volume,
			Amount: k.Amount,
		})
	}

	r.cache.Set(cacheKey, results, 30*time.Second)
	return results, nil
}

// GetFollowedStocks 获取关注列表
func (r *StockRepositoryImpl) GetFollowedStocks(userId string) ([]stock.FollowedStock, error) {
	var results []stock.FollowedStock
	err := r.db.Where("user_id = ?", userId).Order("sort ASC").Find(&results).Error
	return results, err
}

// FollowStock 关注股票（含软删除恢复与名称/备注更新）
func (r *StockRepositoryImpl) FollowStock(userId, code, name, note string) error {
	var existing stock.FollowedStock
	err := r.db.Unscoped().Where("user_id = ? AND stock_code = ?", userId, code).First(&existing).Error
	if err == nil {
		if existing.DeletedAt.Valid {
			return r.db.Unscoped().Model(&existing).Updates(map[string]interface{}{
				"deleted_at": nil,
				"stock_name": name,
				"note":       note,
			}).Error
		}
		updates := map[string]interface{}{}
		if name != "" && name != existing.StockName {
			updates["stock_name"] = name
		}
		if note != existing.Note {
			updates["note"] = note
		}
		if len(updates) > 0 {
			return r.db.Model(&existing).Updates(updates).Error
		}
		return nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return err
	}

	var maxSort int
	r.db.Model(&stock.FollowedStock{}).Where("user_id = ?", userId).Select("COALESCE(MAX(sort), 0)").Scan(&maxSort)

	return r.db.Create(&stock.FollowedStock{
		StockCode: code,
		StockName: name,
		UserId:    userId,
		Sort:      maxSort + 1,
		Note:      note,
	}).Error
}

// UnfollowStock 取消关注
func (r *StockRepositoryImpl) UnfollowStock(userId, code string) error {
	return r.db.Where("user_id = ? AND stock_code = ?", userId, code).Delete(&stock.FollowedStock{}).Error
}

// UpdateCost 更新成本价
func (r *StockRepositoryImpl) UpdateCost(userId, code string, costPrice, quantity float64) error {
	return r.db.Model(&stock.FollowedStock{}).
		Where("user_id = ? AND stock_code = ?", userId, code).
		Updates(map[string]interface{}{
			"cost_price": costPrice,
			"quantity":   quantity,
		}).Error
}

// GetAlarms 获取预警列表
func (r *StockRepositoryImpl) GetAlarms(userId string) ([]stock.StockAlarm, error) {
	var results []stock.StockAlarm
	err := r.db.Where("user_id = ?", userId).Find(&results).Error
	return results, err
}

// SetAlarm 设置预警
func (r *StockRepositoryImpl) SetAlarm(userId string, alarm *stock.StockAlarm) error {
	var existing stock.StockAlarm
	err := r.db.Where("user_id = ? AND stock_code = ?", userId, alarm.StockCode).First(&existing).Error
	if err == nil {
		existing.HighPrice = alarm.HighPrice
		existing.LowPrice = alarm.LowPrice
		existing.Enabled = alarm.Enabled
		return r.db.Save(&existing).Error
	}
	alarm.UserId = userId
	return r.db.Create(alarm).Error
}

// DeleteAlarm 删除预警
func (r *StockRepositoryImpl) DeleteAlarm(userId string, id uint) error {
	return r.db.Where("user_id = ? AND id = ?", userId, id).Delete(&stock.StockAlarm{}).Error
}

// GetMarkets 获取市场列表
func (r *StockRepositoryImpl) GetMarkets() ([]string, error) {
	var markets []string
	err := r.db.Model(&stock.StockInfo{}).Distinct("market").Pluck("market", &markets).Error
	return markets, err
}

// GetIndustries 获取行业列表
func (r *StockRepositoryImpl) GetIndustries() ([]string, error) {
	var industries []string
	err := r.db.Model(&stock.StockInfo{}).Distinct("industry").Where("industry != ''").Pluck("industry", &industries).Error
	return industries, err
}

// GetConcepts 获取概念列表
func (r *StockRepositoryImpl) GetConcepts() ([]string, error) {
	var concepts []string
	err := r.db.Model(&stock.StockInfo{}).Distinct("concept").Where("concept != ''").Pluck("concept", &concepts).Error
	return concepts, err
}

// 以下是暂时的空实现，后续完善

func (r *StockRepositoryImpl) GetMoneyHistory(code string) ([]stock.MoneyFlowInfo, error) {
	return nil, nil
}

func (r *StockRepositoryImpl) GetMoneyTrend(code string) ([]stock.MoneyFlowInfo, error) {
	return nil, nil
}

func (r *StockRepositoryImpl) GetFinancialInfo(code string) (*stock.FinancialInfo, error) {
	return nil, nil
}

func (r *StockRepositoryImpl) GetConceptInfo(code string) ([]string, error) {
	return nil, nil
}

func (r *StockRepositoryImpl) GetHolderNum(code string) (int, error) {
	return 0, nil
}

func (r *StockRepositoryImpl) GetRZRQ(code string) ([]stock.RZRQInfo, error) {
	return nil, nil
}

func (r *StockRepositoryImpl) SelectStocks(criteria *stock.StockSelectionCriteria) (*stock.StockSelectionResult, error) {
	return nil, nil
}

func (r *StockRepositoryImpl) GetAllStockInfo(criteria *stock.StockSelectionCriteria, page, pageSize int) ([]stock.AllStockInfo, int64, error) {
	return nil, 0, nil
}

func (r *StockRepositoryImpl) GetHotStrategies() ([]stock.HotStrategyData, error) {
	return nil, nil
}

// ========== 选股API实现 ==========

// GetAllMarketQuotes 获取全市场行情
func (r *StockRepositoryImpl) GetAllMarketQuotes() ([]*stock.MarketQuote, error) {
	cacheKey := cache.GenerateKey("picker", "all-quotes")
	var cached []*stock.MarketQuote

	if r.cache.Get(cacheKey, &cached) {
		return cached, nil
	}

	quotes, err := r.emClient.GetAllAShareQuotes()
	if err != nil {
		return nil, err
	}

	result := make([]*stock.MarketQuote, 0, len(quotes))
	for _, q := range quotes {
		result = append(result, &stock.MarketQuote{
			Code:                 q.Code,
			Name:                 q.Name,
			Price:                q.Price,
			ChangePercent:        q.ChangePercent,
			Change:               q.Change,
			Open:                 q.Open,
			High:                 q.High,
			Low:                  q.Low,
			PrevClose:            q.PrevClose,
			Volume:               q.Volume,
			Amount:               q.Amount,
			TurnoverRate:         q.TurnoverRate,
			VolumeRatio:          q.VolumeRatio,
			CirculatingMarketCap: q.CirculatingMarketCap,
			TotalMarketCap:       q.TotalMarketCap,
			Pe:                   q.Pe,
			Pb:                   q.Pb,
			Market:               q.Market,
		})
	}

	r.cache.Set(cacheKey, result, 5*time.Second)
	return result, nil
}

// GetTodayTimeline 获取今日分时数据
func (r *StockRepositoryImpl) GetTodayTimeline(code string) ([]stock.TimelinePoint, error) {
	cacheKey := cache.GenerateKey("picker", "timeline", code)
	var cached []stock.TimelinePoint

	if r.cache.Get(cacheKey, &cached) {
		return cached, nil
	}

	// 使用东方财富1分钟K线作为分时数据
	klineData, _, err := r.emClient.GetKLine(code, eastmoney.KLineType1Min, "", 240)
	if err != nil {
		return nil, err
	}

	result := make([]stock.TimelinePoint, 0, len(klineData))
	var totalAmount, totalVolume float64

	for _, k := range klineData {
		if k.Date == "" {
			continue
		}
		timeStr := k.Date
		if len(k.Date) > 10 {
			timeStr = k.Date[11:]
		}

		totalAmount += k.Amount
		totalVolume += float64(k.Volume)

		avgPrice := 0.0
		if totalVolume > 0 {
			avgPrice = totalAmount / totalVolume / 100.0
		}

		result = append(result, stock.TimelinePoint{
			Time:     timeStr,
			Price:    k.Close,
			AvgPrice: avgPrice,
		})
	}

	r.cache.Set(cacheKey, result, 10*time.Second)
	return result, nil
}

// EndOfDayPicker 尾盘选股
func (r *StockRepositoryImpl) EndOfDayPicker(req *stock.EndOfDayPickerRequest) (*stock.PickerResponse, error) {
	allQuotes, err := r.GetAllMarketQuotes()
	if err != nil {
		return nil, err
	}

	var results []stock.PickerStockResult

	for _, quote := range allQuotes {
		// 排除ST
		if req.ExcludeST && (strings.Contains(quote.Name, "ST") || strings.Contains(quote.Name, "*ST")) {
			continue
		}

		// 市值筛选
		if quote.CirculatingMarketCap < req.MarketCapMin || quote.CirculatingMarketCap > req.MarketCapMax {
			continue
		}

		// 量比筛选
		if quote.VolumeRatio < req.VolumeRatioMin {
			continue
		}

		// 涨幅筛选
		if quote.ChangePercent < req.ChangePercentMin || quote.ChangePercent > req.ChangePercentMax {
			continue
		}

		// 换手率筛选
		if quote.TurnoverRate < req.TurnoverRateMin || quote.TurnoverRate > req.TurnoverRateMax {
			continue
		}

		result := stock.PickerStockResult{
			Code:                 quote.Code,
			Name:                 quote.Name,
			Price:                quote.Price,
			ChangePercent:        quote.ChangePercent,
			Change:               quote.Change,
			Volume:               quote.Volume,
			Amount:               quote.Amount,
			TurnoverRate:         quote.TurnoverRate,
			VolumeRatio:          quote.VolumeRatio,
			CirculatingMarketCap: quote.CirculatingMarketCap,
			TotalMarketCap:       quote.TotalMarketCap,
			Pe:                   quote.Pe,
			Pb:                   quote.Pb,
			High:                 quote.High,
			Low:                  quote.Low,
			Open:                 quote.Open,
			PrevClose:            quote.PrevClose,
			Market:               quote.Market,
		}

		results = append(results, result)
	}

	// 第二阶段：获取分时数据并计算分时强度
	var finalResults []stock.PickerStockResult
	for _, result := range results {
		fullCode := result.Code
		if result.Market == "SH" {
			fullCode = "sh" + result.Code
		} else if result.Market == "SZ" {
			fullCode = "sz" + result.Code
		} else if result.Market == "BJ" {
			fullCode = "bj" + result.Code
		}

		timeline, err := r.GetTodayTimeline(fullCode)
		if err != nil || len(timeline) == 0 {
			continue
		}

		// 计算分时强度
		aboveCount := 0
		for _, p := range timeline {
			if p.Price >= p.AvgPrice && p.AvgPrice > 0 {
				aboveCount++
			}
		}
		ratio := float64(aboveCount) / float64(len(timeline)) * 100

		if ratio >= req.TimelineAboveAvgRatio {
			result.Timeline = timeline
			result.TimelineAboveAvgRatio = ratio
			finalResults = append(finalResults, result)
		}
	}

	// 按时分强度排序
	for i := range finalResults {
		for j := i + 1; j < len(finalResults); j++ {
			if finalResults[j].TimelineAboveAvgRatio > finalResults[i].TimelineAboveAvgRatio {
				finalResults[i], finalResults[j] = finalResults[j], finalResults[i]
			}
		}
	}

	return &stock.PickerResponse{
		Strategy: stock.PickerStrategyEndOfDay,
		Total:    len(finalResults),
		List:     finalResults,
	}, nil
}

// MomentumPicker 妖股候选人扫描
func (r *StockRepositoryImpl) MomentumPicker(req *stock.MomentumPickerRequest) (*stock.PickerResponse, error) {
	allQuotes, err := r.GetAllMarketQuotes()
	if err != nil {
		return nil, err
	}

	var results []stock.PickerStockResult

	for _, quote := range allQuotes {
		// 排除ST
		if req.ExcludeST && (strings.Contains(quote.Name, "ST") || strings.Contains(quote.Name, "*ST")) {
			continue
		}

		// 市值筛选
		if quote.CirculatingMarketCap < req.MarketCapMin || quote.CirculatingMarketCap > req.MarketCapMax {
			continue
		}

		// 价格筛选
		if quote.Price < req.PriceMin || quote.Price > req.PriceMax {
			continue
		}

		// 换手率筛选
		if quote.TurnoverRate < req.AvgTurnoverMin {
			continue
		}

		result := stock.PickerStockResult{
			Code:                 quote.Code,
			Name:                 quote.Name,
			Price:                quote.Price,
			ChangePercent:        quote.ChangePercent,
			Change:               quote.Change,
			Volume:               quote.Volume,
			Amount:               quote.Amount,
			TurnoverRate:         quote.TurnoverRate,
			VolumeRatio:          quote.VolumeRatio,
			CirculatingMarketCap: quote.CirculatingMarketCap,
			TotalMarketCap:       quote.TotalMarketCap,
			Pe:                   quote.Pe,
			Pb:                   quote.Pb,
			High:                 quote.High,
			Low:                  quote.Low,
			Open:                 quote.Open,
			PrevClose:            quote.PrevClose,
			Market:               quote.Market,
			AvgTurnover5d:        quote.TurnoverRate,
		}

		results = append(results, result)
	}

	// 第二阶段：获取K线数据计算技术指标
	var finalResults []stock.PickerStockResult

	type klineResult struct {
		idx   int
		kline *stock.KLineData
	}

	sem := make(chan struct{}, 8)
	var wg sync.WaitGroup
	var mu sync.Mutex

	for i := range results {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			quote := results[idx]
			fullCode := quote.Code
			if quote.Market == "SH" {
				fullCode = "sh" + quote.Code
			} else if quote.Market == "SZ" {
				fullCode = "sz" + quote.Code
			}

			kline, err := r.GetKLine(fullCode, "day", 80, "qfq")
			if err != nil || kline == nil || len(kline.List) < 60 {
				return
			}

			mu.Lock()
			finalResults = append(finalResults, results[idx])
			lastIdx := len(finalResults) - 1
			finalResults[lastIdx].Ma60 = 0
			finalResults[lastIdx].Ma60Distance = 0
			finalResults[lastIdx].MomentumRatio = 0
			finalResults[lastIdx].High20d = 0
			finalResults[lastIdx].Low20d = 0

			klines := kline.List
			if len(klines) >= 60 {
				// 计算60日均线
				ma60 := 0.0
				for _, k := range klines[len(klines)-60:] {
					ma60 += k.Close
				}
				ma60 /= 60
				finalResults[lastIdx].Ma60 = ma60
				finalResults[lastIdx].Ma60Distance = (quote.Price - ma60) / ma60 * 100

				// 趋势因子筛选
				if req.TrendAboveMA60 && quote.Price < ma60 {
					finalResults = finalResults[:len(finalResults)-1]
					mu.Unlock()
					return
				}
			}

			if len(klines) >= 20 {
				// 计算20日高低和动量
				high20d := 0.0
				low20d := 1e18
				for _, k := range klines[len(klines)-20:] {
					if k.High > high20d {
						high20d = k.High
					}
					if k.Low < low20d && k.Low > 0 {
						low20d = k.Low
					}
				}
				finalResults[lastIdx].High20d = high20d
				finalResults[lastIdx].Low20d = low20d
				if low20d > 0 {
					momentumRatio := (high20d - low20d) / low20d * 100
					finalResults[lastIdx].MomentumRatio = momentumRatio

					// 动量因子筛选
					if momentumRatio < req.MomentumThreshold {
						finalResults = finalResults[:len(finalResults)-1]
						mu.Unlock()
						return
					}
				}
			}

			mu.Unlock()
		}(i)
	}

	wg.Wait()

	// 按动量强度排序
	for i := range finalResults {
		for j := i + 1; j < len(finalResults); j++ {
			if finalResults[j].MomentumRatio > finalResults[i].MomentumRatio {
				finalResults[i], finalResults[j] = finalResults[j], finalResults[i]
			}
		}
	}

	return &stock.PickerResponse{
		Strategy: stock.PickerStrategyMomentum,
		Total:    len(finalResults),
		List:     finalResults,
	}, nil
}

// KunpengPicker 鲲鹏战法筛选
func (r *StockRepositoryImpl) KunpengPicker(req *stock.KunpengPickerRequest) (*stock.PickerResponse, error) {
	allQuotes, err := r.GetAllMarketQuotes()
	if err != nil {
		return nil, err
	}

	var results []stock.PickerStockResult

	for _, quote := range allQuotes {
		// 排除ST
		if req.ExcludeST && (strings.Contains(quote.Name, "ST") || strings.Contains(quote.Name, "*ST")) {
			continue
		}

		// 市值筛选
		if quote.TotalMarketCap < req.MarketCapMin || quote.TotalMarketCap > req.MarketCapMax {
			continue
		}

		// PE筛选
		if quote.Pe <= req.PeMin || quote.Pe > req.PeMax {
			continue
		}

		// 价格筛选
		if quote.Price < req.PriceMin || quote.Price > req.PriceMax {
			continue
		}

		// 计算净利润（通过PE和市值估算）
		netProfit := quote.TotalMarketCap / quote.Pe
		if netProfit < req.NetProfitMin {
			continue
		}

		// 计算安全评分
		marketCapScore := math.Max(0, 30-math.Abs(quote.TotalMarketCap-200)/10)
		profitScore := math.Min(30, (netProfit/5)*30)
		peScore := 0.0
		if quote.Pe > 0 {
			peScore = math.Max(0, 25-(quote.Pe-15)/2)
		}
		priceScore := 15.0
		if quote.Price < 5 || quote.Price > 50 {
			priceScore = 10
			if quote.Price < 3 || quote.Price > 80 {
				priceScore = 5
			}
		}
		safetyScore := math.Min(100, math.Max(0, marketCapScore+profitScore+peScore+priceScore))

		// 计算潜在倍数
		potentialMarketCap := netProfit * 50
		maxPotentialMarketCap := math.Min(potentialMarketCap, 1000)
		potentialMultiple := maxPotentialMarketCap / quote.TotalMarketCap

		result := stock.PickerStockResult{
			Code:                 quote.Code,
			Name:                 quote.Name,
			Price:                quote.Price,
			ChangePercent:        quote.ChangePercent,
			Change:               quote.Change,
			Volume:               quote.Volume,
			Amount:               quote.Amount,
			TurnoverRate:         quote.TurnoverRate,
			VolumeRatio:          quote.VolumeRatio,
			CirculatingMarketCap: quote.CirculatingMarketCap,
			TotalMarketCap:       quote.TotalMarketCap,
			Pe:                   quote.Pe,
			Pb:                   quote.Pb,
			High:                 quote.High,
			Low:                  quote.Low,
			Open:                 quote.Open,
			PrevClose:            quote.PrevClose,
			Market:               quote.Market,
			NetProfit:            netProfit,
			SafetyScore:          safetyScore,
			PotentialMultiple:    potentialMultiple,
		}

		results = append(results, result)
	}

	// 按安全评分排序
	for i := range results {
		for j := i + 1; j < len(results); j++ {
			if results[j].SafetyScore > results[i].SafetyScore {
				results[i], results[j] = results[j], results[i]
			}
		}
	}

	return &stock.PickerResponse{
		Strategy: stock.PickerStrategyKunpeng,
		Total:    len(results),
		List:     results,
	}, nil
}
