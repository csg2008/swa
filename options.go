package main

import (
	"encoding/json"
	"errors"
	"path/filepath"
	"strings"
)

// ErrNeedValidateAuth 需要检查账号授权
var ErrNeedValidateAuth = errors.New("need validate auth")

// ErrFSWatcherStop 单一窗口回执目录事件监听停止
var ErrFSWatcherStop = errors.New("file system watcher stopped")

// DebugLevel 调试级别
type DebugLevel struct {
	Value int
	Label string
}

// Counter 计数器
type Counter struct {
	Error    uint64
	Upload   uint64
	Download uint64
}

// Options 配置选项
type Options struct {
	Status     bool     `json:"-" label:"连接状态"`
	Debug      int      `json:"debug" label:"调试级别"`
	Timeout    int      `json:"timeout" label:"通信超时时间"`
	Interval   int      `json:"interval" label:"轮询远程服务器数据时间间隔"`
	TimeLag    int      `json:"time_lag" label:"本身与远程服务器时间差间隔"`
	ECid       string   `json:"ecid" label:"企业身份ID"`
	UID        string   `json:"uid" label:"用户ID"`
	UName      string   `json:"uname" label:"用户名"`
	Pwd        string   `json:"-" label:"账号密码"`
	Token      string   `json:"token" label:"数据签名 Token"`
	URL        string   `json:"url" label:"数通天下快捷报关服务器 URL"`
	DataPath   string   `json:"data_path" label:"单一窗口数据目录"`
	Counter    *Counter `json:"-" label:"计数器"`
	configFile string   `label:"配置文件路径"`
	oldUName   string   `label:"旧用户名"`
}

// Init 初始化配置选项
func (opt *Options) Init() {
	opt.configFile = GetAppPath() + "/config.json"

	opt.Load()
	if 0 == opt.Timeout {
		opt.Timeout = 10
	}

	if 0 == opt.Interval {
		opt.Interval = 300
	}

	if 0 == opt.TimeLag {
		opt.TimeLag = 300
	}

	if "" == opt.DataPath {
		opt.DataPath = "C:\\ImpPath"
	}

	// 先保存一个用户名副本，如果修改了用户名就验证密码
	opt.oldUName = opt.UName

	// 初始化计数器
	opt.Counter = new(Counter)
}

// Validate 验证选项数据是否正确
func (opt *Options) Validate() error {
	var err error

	if !strings.HasPrefix(opt.URL, "http://") && !strings.HasPrefix(opt.URL, "https://") {
		err = errors.New("服务器 URL 必须以 http:// 或 https:// 开头")
	} else if p, e := filepath.Abs(opt.DataPath); nil != e || "" == p {
		err = errors.New("单一窗口数据目录不存在或不可读写")
	} else if "" == opt.UName {
		err = errors.New("用户名不能为空")
	} else if "" == opt.Pwd && opt.oldUName != opt.UName {
		err = errors.New("登录密码不能为空")
	}

	if nil == err && "" != opt.Pwd {
		err = ErrNeedValidateAuth
	}

	return err
}

// Load 从文件加载配置
func (opt *Options) Load() error {
	var data, err = FileGetContents(opt.configFile)
	if nil == err && nil != data {
		err = json.Unmarshal(data, opt)
	}

	return err
}

// Save 保存配置到文件
func (opt *Options) Save() error {
	var data, err = json.Marshal(opt)
	if nil == err && nil != data {
		err = FilePutContents(opt.configFile, data, false)
	}

	return err
}
