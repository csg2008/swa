package main

import (
	"errors"
	"io/ioutil"
	"net/http"
	"net/url"
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/PuerkitoBio/goquery"
	"github.com/clbanning/mxj"
	"github.com/fsnotify/fsnotify"
)

// Message 远程消息格式
type Message struct {
	Code int         `json:"code" label:"消息状态码，0 表示失败 1 表示成功"`
	Msg  string      `json:"msg" label:"消息提示内容"`
	Time string      `json:"time" label:"消息时间"`
	Data interface{} `json:"data" label:"返回的数据"`
}

// Execute 指令执行器
type Execute struct {
	client  *Client
	options *Options
	mux     *sync.Mutex
	tip     func(category string, level int, msg ...string)
	counter func(c *Counter)
	quit    chan struct{}
	failed  map[string]int
}

// Init 初始化指令执行器
func (exe *Execute) Init(opt *Options, tip func(category string, level int, msg ...string), c func(c *Counter)) {
	exe.tip = tip
	exe.counter = c
	exe.options = opt

	exe.mux = new(sync.Mutex)
	exe.failed = make(map[string]int)
	exe.client = NewClient(exe.options.Debug, tip)
}

// Start 开始业务指令循环
func (exe *Execute) Start() {
	exe.mux.Lock()
	defer exe.mux.Unlock()

	if !exe.options.Status {
		exe.options.Status = true
		exe.quit = make(chan struct{})

		go exe.watcher()
		go exe.consume()
	}
}

// Stop 停止业务指令循环
func (exe *Execute) Stop() {
	exe.mux.Lock()
	defer exe.mux.Unlock()

	if exe.options.Status {
		exe.options.Status = false

		exe.quit <- struct{}{}
		exe.quit <- struct{}{}

		close(exe.quit)
	}
}

// Auth 账号授权检查
func (exe *Execute) Auth() error {
	var token string
	var url = exe.options.URL + "admin/index/login"
	var doc, err = exe.client.GetDoc(url, nil)
	if nil == err {
		doc.Find("form#login-form input").Each(func(i int, s *goquery.Selection) {
			if n, ok := s.Attr("name"); ok && "__token__" == n {
				token, _ = s.Attr("value")
			}
		})
	}

	if "" == token {
		err = errors.New("读取远程登录表单 token 失败")
	} else {
		var flag = true
		var msg = &Message{}
		var payload = &ClientPayload{KeepAlive: true, Method: "POST", Data: "__token__=" + token + "&username=" + exe.options.UName + "&password=" + exe.options.Pwd}
		var header = make(http.Header)

		header.Set("X-Requested-With", "XMLHttpRequest")
		payload.Header = &header
		err = exe.client.GetCodec(url, payload, "json", msg)
		if nil == err && 1 == msg.Code && nil != msg.Data {
			if v, ok := msg.Data.(map[string]interface{}); ok && nil != v {
				if id, ok := v["id"].(string); ok && "" != id {
					exe.options.UID = id
				} else if id, ok := v["id"].(int64); ok && id > 0 {
					exe.options.UID = strconv.FormatInt(id, 10)
				} else if id, ok := v["id"].(float64); ok && int64(id) > 0 {
					exe.options.UID = strconv.FormatInt(int64(id), 10)
				} else {
					flag = false
				}
				if ecid, ok := v["ecid"].(string); ok && "" != ecid {
					exe.options.ECid = ecid
				} else if ecid, ok := v["ecid"].(int64); ok && ecid > 0 {
					exe.options.ECid = strconv.FormatInt(ecid, 10)
				} else if ecid, ok := v["ecid"].(float64); ok && int64(ecid) > 0 {
					exe.options.ECid = strconv.FormatInt(int64(ecid), 10)
				} else {
					flag = false
				}
			}
		}

		if !flag {
			err = errors.New("登录失败，请检查用户名与密码后继续")
		}
	}

	return err
}

// watcher 监视本地指定目录的文件变化事件
func (exe *Execute) watcher() {
	var i int
	var e fsnotify.Event
	var fw, err = fsnotify.NewWatcher()
	if nil == err {
		if files, err := ioutil.ReadDir(exe.options.DataPath); nil == err && nil != files {
			for _, file := range files {
				if file.IsDir() && !strings.HasPrefix(file.Name(), ".") {
					var v = exe.options.DataPath + "/" + file.Name() + "/InBox"
					if IsDir(v) {
						i++
						fw.Add(v)
					}
				}
			}
		}

		if i > 0 {
			for {
				select {
				case e = <-fw.Events:
					if IsFile(e.Name) {
						err = exe.upload(e.Name)
					}
				case err = <-fw.Errors:

				case <-exe.quit:
					err = ErrFSWatcherStop
				}

				if err == ErrFSWatcherStop {
					break
				} else if nil != err {
					atomic.AddUint64(&exe.options.Counter.Error, 1)
					exe.tip("notify", 2, "", "报文处理出错："+err.Error())
				} else {
					atomic.AddUint64(&exe.options.Counter.Upload, 1)
				}
			}
		} else {
			exe.tip("error", 1, "", "没有发现需要监听的目录，请确认选择的单一窗口客户端数据目录是否正确？")
		}
	}
}

// upload 上传回执到远程服务器
func (exe *Execute) upload(file string) error {
	var err error
	var t = strings.Replace(strings.ToLower(file), "\\", "/", -1)
	if strings.HasSuffix(t, ".xml") {
		var content []byte

		if content, err = exe.getFile(file); nil == err && nil != content {
			var v = strings.Split(t, "/")
			var vv = strings.Split(v[len(v)-1], "_")
			var param = map[string]string{
				"ecid":     exe.options.ECid,
				"admin_id": exe.options.UID,
				"action":   "result",
				"file":     v[len(v)-1],
				"content":  string(content),
			}

			if "receipt" == vv[0] {
				param["id"] = "0"
				param["action"] = "receipt"
				param["original_bn"] = vv[1]
			} else if "successed" == vv[0] || "failed" == vv[0] {
				param["action"] = "status"
				param["id"] = strings.Split(strings.Split(v[len(v)-1], ".")[1], "(")[0]
			} else {
				param["action"] = "other"
			}

			err = exe.receipt(param)
		}
	}

	return err
}

// getFileMap 读取回执 XML 文件并解析为 JSON
func (exe *Execute) getFile(file string) ([]byte, error) {
	var c, err = FileGetContents(file)
	if nil == err && nil != c {
		var m mxj.Map
		m, err = mxj.NewMapXml(c)
		if nil == err {
			return m.Json()
		}
	}

	return nil, err
}

// consume 消费服务器端命令
func (exe *Execute) consume() {
	var t = time.NewTicker(time.Duration(exe.options.Interval) * time.Second)

	for {
		select {
		case <-t.C:
			exe.consumeRemoteCommand()
		case <-exe.quit:
			return
		}
	}
}

// consumeRemoteCommand 消费服务器端的命令
func (exe *Execute) consumeRemoteCommand() {
	var msg = &Message{}
	var url = exe.options.URL + "api/Chinaport/Commands"
	var param = map[string]string{"ecid": exe.options.ECid, "admin_id": exe.options.UID}
	var payload = &ClientPayload{KeepAlive: true, Method: "POST", Data: exe.mapToQS(param)}
	var err = exe.client.GetCodec(url, payload, "json", msg)

	if nil == err {
		if 1 == msg.Code && nil != msg.Data {
			if rows, ok := msg.Data.([]interface{}); ok && len(rows) > 0 {
				var category []string

				for _, v := range rows {
					if row, ok := v.(map[string]interface{}); ok {
						var args = map[string]string{
							"id":       exe.numToStr(row["id"]),
							"ecid":     exe.options.ECid,
							"admin_id": exe.options.UID,
						}

						category = strings.SplitN(row["category"].(string), "|", 2)
						switch category[0] {
						case "xml":
							err = exe.download(args)
							if nil != err {
								exe.tip("notify", 3, "", err.Error())
							}
						default:
							err = errors.New("未知的命令：" + row["category"].(string))
							exe.tip("notify", 4, "", err.Error())
						}

						if nil != err {
							atomic.AddUint64(&exe.options.Counter.Error, 1)

							// 如果命令执行失败三次，就不要重复执行了并直接上报执行失败，上报成功就清除失败标志
							exe.failed[args["id"]] = exe.failed[args["id"]] + 1
							if exe.failed[args["id"]] >= 3 {
								args["action"] = "download"
								args["status"] = "failed"

								if err = exe.receipt(args); nil == err {
									delete(exe.failed, args["id"])
								}
							}
						} else {
							atomic.AddUint64(&exe.options.Counter.Download, 1)
						}
					}
				}

				exe.counter(exe.options.Counter)
			} else {
				exe.tip("notify", 4, "", "从远程服务器读取的命令列表为空")
			}
		} else if 0 == msg.Code && "" != msg.Msg {
			exe.tip("notify", 2, "", "从远程服务器获取命令出错："+msg.Msg)
		}
	} else {
		exe.tip("notify", 2, "", "从远程服务器获取命令出错："+err.Error())
	}
}

// receipt 状态回传
func (exe *Execute) receipt(param map[string]string) error {
	var msg = &Message{}
	var url = exe.options.URL + "api/Chinaport/Receipt"
	var payload = &ClientPayload{KeepAlive: true, Method: "POST", Data: exe.mapToQS(param)}
	var err = exe.client.GetCodec(url, payload, "json", msg)

	if nil == err && 0 == msg.Code {
		err = errors.New(msg.Msg)
	}

	return err
}

// download 下载数据
func (exe *Execute) download(param map[string]string) error {
	var msg = &Message{}
	var url = exe.options.URL + "api/Chinaport/Download"
	var payload = &ClientPayload{KeepAlive: true, Method: "POST", Data: exe.mapToQS(param)}
	var err = exe.client.GetCodec(url, payload, "json", msg)

	if nil == err {
		if 1 == msg.Code && nil != msg.Data {
			if data, ok := msg.Data.(map[string]interface{}); ok && nil != data {
				var file = exe.options.DataPath + "/" + data["path"].(string)
				err = FilePutContents(file, []byte(data["xml"].(string)), false)
				if nil == err {
					param["action"] = "download"
					param["status"] = "ok"

					err = exe.receipt(param)
				}
			}
		} else {
			if "" != msg.Msg {
				err = errors.New(msg.Msg)
			} else {
				err = errors.New("未知错误")
			}
		}
	}

	return err
}

// mapToQS 将 map 结构的参数转换为 form 表单字符串形式
func (exe *Execute) mapToQS(data map[string]string) string {
	var v = make(url.Values)

	for k1, v1 := range data {
		v.Set(k1, v1)
	}

	return v.Encode()
}

// numToStr 数值转字符串
func (exe *Execute) numToStr(input interface{}) string {
	var v string

	if val, ok := input.(float64); ok {
		v = strconv.FormatInt(int64(val), 10)
	} else if val, ok := input.(int64); ok {
		v = strconv.FormatInt(val, 10)
	} else if val, ok := input.(string); ok {
		v = val
	}

	return v
}
