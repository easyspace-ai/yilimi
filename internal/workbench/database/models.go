package database

import (
	"time"
)

// Notebook 笔记本模型
type Notebook struct {
	ID      string    `json:"id" db:"id"`
	Name    string    `json:"name" db:"name"`
	Icon    string    `json:"icon" db:"icon"`
	Sort    int       `json:"sort" db:"sort"`
	Closed  bool      `json:"closed" db:"closed"`
	Created time.Time `json:"created" db:"created"`
	Updated time.Time `json:"updated" db:"updated"`
}

// Block 块模型 - 核心数据结构
type Block struct {
	ID       string    `json:"id" db:"id"`
	ParentID string    `json:"parentID" db:"parent_id"`
	RootID   string    `json:"rootID" db:"root_id"`
	Type     string    `json:"type" db:"type"`
	SubType  string    `json:"subType" db:"subtype"`
	Content  string    `json:"content" db:"content"`
	Markdown string    `json:"markdown" db:"markdown"`
	Path     string    `json:"path" db:"path"`
	HPath    string    `json:"hPath" db:"hpath"`
	Name     string    `json:"name" db:"name"`
	Alias    string    `json:"alias" db:"alias"`
	Memo     string    `json:"memo" db:"memo"`
	Tag      string    `json:"tag" db:"tag"`
	IAL      string    `json:"ial" db:"ial"`
	Sort     int       `json:"sort" db:"sort"`
	Created  time.Time `json:"created" db:"created"`
	Updated  time.Time `json:"updated" db:"updated"`
	Box      string    `json:"box" db:"box"` // 笔记本ID，用于关联
}

// Attribute 属性模型
type Attribute struct {
	ID      int       `json:"id" db:"id"`
	BlockID string    `json:"blockID" db:"block_id"`
	Name    string    `json:"name" db:"name"`
	Value   string    `json:"value" db:"value"`
	Type    string    `json:"type" db:"type"`
	Created time.Time `json:"created" db:"created"`
	Updated time.Time `json:"updated" db:"updated"`
}

// Ref 引用关系模型
type Ref struct {
	ID              string    `json:"id" db:"id"`
	DefBlockID      string    `json:"defBlockID" db:"def_block_id"`
	DefBlockPath    string    `json:"defBlockPath" db:"def_block_path"`
	DefBlockContent string    `json:"defBlockContent" db:"def_block_content"`
	DefBlockSubType string    `json:"defBlockSubType" db:"def_block_subtype"`
	BlockID         string    `json:"blockID" db:"block_id"`
	RootID          string    `json:"rootID" db:"root_id"`
	Box             string    `json:"box" db:"box"`
	Path            string    `json:"path" db:"path"`
	Content         string    `json:"content" db:"content"`
	Markdown        string    `json:"markdown" db:"markdown"`
	Type            string    `json:"type" db:"type"`
	Created         time.Time `json:"created" db:"created"`
	Updated         time.Time `json:"updated" db:"updated"`
}

// Span 全文搜索索引模型
type Span struct {
	ID       int       `json:"id" db:"id"`
	BlockID  string    `json:"blockID" db:"block_id"`
	RootID   string    `json:"rootID" db:"root_id"`
	Box      string    `json:"box" db:"box"`
	Path     string    `json:"path" db:"path"`
	Content  string    `json:"content" db:"content"`
	Markdown string    `json:"markdown" db:"markdown"`
	Type     string    `json:"type" db:"type"`
	IAL      string    `json:"ial" db:"ial"`
	Created  time.Time `json:"created" db:"created"`
	Updated  time.Time `json:"updated" db:"updated"`
}

// FileTree 文件树节点模型（用于API响应）
type FileTree struct {
	ID       string      `json:"id"`
	Name     string      `json:"name"`
	Type     string      `json:"type"`
	SubType  string      `json:"subType,omitempty"`
	Path     string      `json:"path"`
	Icon     string      `json:"icon"`
	Count    int         `json:"count"`
	Created  time.Time   `json:"created"`
	Updated  time.Time   `json:"updated"`
	Children []*FileTree `json:"children,omitempty"`
}

// SearchResult 搜索结果模型
type SearchResult struct {
	Blocks            []*Block `json:"blocks"`
	MatchedBlockCount int      `json:"matchedBlockCount"`
	MatchedRootCount  int      `json:"matchedRootCount"`
	PageCount         int      `json:"pageCount"`
}

// Tag 标签模型
type Tag struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// APIResponse 统一API响应格式
type APIResponse struct {
	Code int         `json:"code"`
	Msg  string      `json:"msg"`
	Data interface{} `json:"data,omitempty"`
}

// WSMessage WebSocket消息格式
type WSMessage struct {
	Cmd      string      `json:"cmd"`
	Data     interface{} `json:"data,omitempty"`
	Callback string      `json:"callback,omitempty"`
}

// Transaction 事务操作
type Transaction struct {
	DoOperations   []*Operation `json:"doOperations"`
	UndoOperations []*Operation `json:"undoOperations"`
}

// Operation 操作记录
type Operation struct {
	Action  string      `json:"action"`
	ID      string      `json:"id"`
	Data    interface{} `json:"data"`
	RetData interface{} `json:"retData"`
}

// BlockType 常量定义
const (
	NodeDocument      = "NodeDocument"
	NodeParagraph     = "NodeParagraph"
	NodeHeading       = "NodeHeading"
	NodeBlockquote    = "NodeBlockquote"
	NodeList          = "NodeList"
	NodeListItem      = "NodeListItem"
	NodeCodeBlock     = "NodeCodeBlock"
	NodeMathBlock     = "NodeMathBlock"
	NodeTable         = "NodeTable"
	NodeThematicBreak = "NodeThematicBreak"
)

// RefType 引用类型常量
const (
	RefTypeRef   = "ref"
	RefTypeEmbed = "embed"
)
