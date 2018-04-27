package main

import (
	"bytes"
	"errors"
	"log"
	"os"
	"strconv"
	"time"

	"github.com/lxn/walk"
	"github.com/lxn/walk/declarative"
)

// UIMainWindow 主窗口主界面
type UIMainWindow struct {
	opt       *Options               `label:"配置选项"`
	exe       *Execute               `label:"指令执行器"`
	log       *os.File               `label:"日志文件"`
	icon      *walk.Icon             `label:"应用主图标"`
	ni        *walk.NotifyIcon       `label:"状态栏提示图标"`
	mw        *walk.MainWindow       `label:"应用主界面窗口"`
	db        *walk.DataBinder       `label:"界面数据源绑定器"`
	ddb       declarative.DataBinder `label:"界面数据源绑定器"`
	sbiDown   *walk.StatusBarItem    `label:"状态栏暂存格子"`
	sbiUpload *walk.StatusBarItem    `label:"状态栏审核格子"`
	sbiError  *walk.StatusBarItem    `label:"状态栏错误格子"`
}

// Init 初始化界面
func (ui *UIMainWindow) Init(opt *Options, exe *Execute) {
	ui.opt = opt
	ui.exe = exe

	ui.mw, _ = walk.NewMainWindow()
	ui.ni, _ = walk.NewNotifyIcon()
	ui.icon, _ = walk.NewIconFromResourceId(1)
	ui.ddb = declarative.DataBinder{
		AssignTo:       &ui.db,
		DataSource:     ui.opt,
		ErrorPresenter: declarative.ToolTipErrorPresenter{},
	}

	// 初始化日志记录
	if ui.opt.Debug > 0 {
		var flag = os.O_WRONLY | os.O_CREATE | os.O_APPEND
		if fp, err := os.OpenFile(GetAppPath()+"/swa.log", flag, os.ModePerm); nil == err {
			ui.log = fp
		}
	}

	ui.SetNotify()
}

// Show 显示主界面
func (ui *UIMainWindow) Show() error {
	var acceptPB, settingPB *walk.PushButton
	var startImg, _ = walk.Resources.Bitmap("3")
	var stopImg, _ = walk.Resources.Bitmap("4")
	var settingImg, _ = walk.Resources.Bitmap("5")
	var aboutImg, _ = walk.Resources.Bitmap("6")

	var err = (declarative.MainWindow{
		AssignTo:   &ui.mw,
		Title:      "数据通天下 - 快捷报关数据传输助理",
		MinSize:    declarative.Size{Width: 600, Height: 300},
		Layout:     declarative.VBox{},
		DataBinder: ui.ddb,
		Children: []declarative.Widget{
			declarative.Composite{
				Layout: declarative.HBox{},
				Children: []declarative.Widget{
					declarative.PushButton{
						AssignTo:       &acceptPB,
						Text:           "开始(&a)",
						Image:          startImg,
						ImageAboveText: true,
						OnClicked: func() {
							var err error

							if "" != ui.opt.ECid && "" != ui.opt.UID {
								if ui.opt.Status {
									ui.exe.Stop()

									if !ui.opt.Status {
										acceptPB.SetText("开始(&a)")
										acceptPB.SetImage(startImg)
										settingPB.SetEnabled(true)
									}
								} else {
									ui.exe.Start()

									if ui.opt.Status {
										acceptPB.SetText("停止(&a)")
										acceptPB.SetImage(stopImg)
										settingPB.SetEnabled(false)
									}
								}
							} else {
								err = errors.New("读取配置数据出错，请单击设置按钮更新配置选项后重试。")
							}

							if nil != err {
								walk.MsgBox(ui.mw, "连接错误", err.Error(), walk.MsgBoxIconInformation)
							}
						},
					},

					declarative.PushButton{
						AssignTo:       &settingPB,
						Text:           "设置(&s)",
						Image:          settingImg,
						ImageAboveText: true,
						OnClicked: func() {
							ui.ShowSetting()
						},
					},

					declarative.PushButton{
						Text:           "关于(&a)",
						Image:          aboutImg,
						ImageAboveText: true,
						OnClicked: func() {
							walk.MsgBox(ui.mw, "关于", "数据通天下 - 快捷报关数据传输助手", walk.MsgBoxIconInformation)
						},
					},
				},
			},
		},
		StatusBarItems: []declarative.StatusBarItem{
			declarative.StatusBarItem{
				AssignTo:    &ui.sbiDown,
				Width:       200,
				Text:        "暂存：0",
				ToolTipText: "暂存到单一窗口客户端报文数量",
			},
			declarative.StatusBarItem{
				AssignTo:    &ui.sbiUpload,
				Width:       200,
				Text:        "审核：0",
				ToolTipText: "从单一窗口客户端接收的审核报文数量",
			},
			declarative.StatusBarItem{
				AssignTo:    &ui.sbiError,
				Width:       200,
				Text:        "错误：0",
				ToolTipText: "传输出错的报文数量",
			},
		},
	}.Create())

	ui.mw.SetIcon(ui.icon)

	return err
}

// SetNotify 设置通知消息
func (ui *UIMainWindow) SetNotify() {
	// Set the icon and a tool tip text.
	if err := ui.ni.SetIcon(ui.icon); err != nil {
		log.Fatal(err)
	}
	if err := ui.ni.SetToolTip("数据通天下 - 快捷报关数据传输助手"); err != nil {
		log.Fatal(err)
	}

	// When the left mouse button is pressed, bring up our balloon.
	ui.ni.MouseDown().Attach(func(x, y int, button walk.MouseButton) {
		if button != walk.LeftButton {
			return
		}
	})

	// We put an exit action into the context menu.
	exitAction := walk.NewAction()
	if err := exitAction.SetText("E&xit"); err != nil {
		log.Fatal(err)
	}
	exitAction.Triggered().Attach(func() { walk.App().Exit(0) })
	if err := ui.ni.ContextMenu().Actions().Add(exitAction); err != nil {
		log.Fatal(err)
	}

	// The notify icon is hidden initially, so we have to make it visible.
	if err := ui.ni.SetVisible(true); err != nil {
		log.Fatal(err)
	}
}

// ShowSetting 显示设置对话框
func (ui *UIMainWindow) ShowSetting() (int, error) {
	var needReLoad bool
	var dlg *walk.Dialog
	var dle *walk.LineEdit
	var icon, _ = walk.NewIconFromResourceId(2)
	var acceptPB, cancelPB *walk.PushButton

	return declarative.Dialog{
		AssignTo:      &dlg,
		Title:         "首选项设置",
		Icon:          icon,
		DefaultButton: &acceptPB,
		CancelButton:  &cancelPB,
		FixedSize:     true,
		DataBinder:    ui.ddb,
		MinSize:       declarative.Size{Width: 550, Height: 400},
		Layout:        declarative.VBox{},
		Children: []declarative.Widget{
			declarative.Composite{
				Layout: declarative.Grid{Columns: 2},
				Children: []declarative.Widget{
					declarative.Label{
						Text: "服务器URL:",
					},
					declarative.LineEdit{
						Text:        declarative.Bind("URL", declarative.SelRequired{}),
						ToolTipText: "数通天下 - 快捷报关服务器地址",
					},

					declarative.Label{
						Text: "用户名:",
					},
					declarative.LineEdit{
						Text:        declarative.Bind("UName", declarative.SelRequired{}),
						ToolTipText: "登录账号名",
					},

					declarative.Label{
						Text: "登录密码:",
					},
					declarative.LineEdit{
						Text:         declarative.Bind("Pwd", declarative.SelRequired{}),
						ToolTipText:  "账号登录密码",
						PasswordMode: true,
					},

					declarative.Label{
						Text: "数据目录:",
					},
					declarative.LineEdit{
						ReadOnly:    true,
						AssignTo:    &dle,
						Text:        declarative.Bind("DataPath", declarative.SelRequired{}),
						ToolTipText: "单击选择单一窗口数据目录",
						OnMouseUp: func(x, y int, button walk.MouseButton) {
							if button == walk.LeftButton {
								var dlg = new(walk.FileDialog)

								dlg.ShowReadOnlyCB = true
								dlg.FilePath = ui.opt.DataPath
								if ok, err := dlg.ShowBrowseFolder(ui.mw); ok && nil == err {
									ui.opt.DataPath = dlg.FilePath

									dle.SetText(ui.opt.DataPath)
								}
							}
						},
					},

					declarative.Label{
						Text: "超时时间:",
					},
					declarative.Slider{
						MinValue:    1,
						MaxValue:    60,
						Value:       declarative.Bind("Timeout"),
						ToolTipText: "与数通天下快捷报关服务器通讯超时时间（单位为秒）",
					},

					declarative.Label{
						Text: "轮询间隔:",
					},
					declarative.Slider{
						MinValue:    1,
						MaxValue:    600,
						Value:       declarative.Bind("Interval"),
						ToolTipText: "从数通天下快捷报关服务器读取数据的间隔时间（单位为秒）",
					},

					declarative.Label{
						Text: "最大时差:",
					},
					declarative.Slider{
						MinValue:    1,
						MaxValue:    600,
						Value:       declarative.Bind("TimeLag"),
						ToolTipText: "数通天下快捷报关服务器时间与当前电脑时间的最大差值（单位为秒）",
					},

					declarative.Label{
						Text: "日志级别:",
					},
					declarative.ComboBox{
						Value:         declarative.Bind("Debug"),
						BindingMember: "Value",
						DisplayMember: "Label",
						Model: []*DebugLevel{
							{0, "关闭"},
							{1, "严重错误"},
							{2, "常规错误"},
							{3, "提示信息"},
							{4, "调试信息"},
						},
					},

					declarative.VSpacer{
						ColumnSpan: 2,
						Size:       8,
					},
				},
			},
			declarative.Composite{
				Layout: declarative.HBox{},
				Children: []declarative.Widget{
					declarative.HSpacer{},
					declarative.PushButton{
						AssignTo: &acceptPB,
						Text:     "保存(&s)",
						OnClicked: func() {
							var err error

							needReLoad = true
							if err = ui.db.Submit(); nil == err {
								if err = ui.opt.Validate(); nil == err {
									err = ui.opt.Save()
								} else if ErrNeedValidateAuth == err {
									if err = ui.exe.Auth(); nil == err {
										err = ui.opt.Save()
									}
								}
							}

							if nil == err {
								dlg.Accept()

								needReLoad = false
							} else {
								walk.MsgBox(ui.mw, "首选项设置", "配置项值检查出错："+err.Error(), walk.MsgBoxIconWarning)
							}
						},
					},
					declarative.PushButton{
						AssignTo: &cancelPB,
						Text:     "取消(&c)",
						OnClicked: func() {
							dlg.Cancel()

							// 如果配置保存失败，需要恢复到之前的状态，以免程序运行出错
							if needReLoad {
								ui.opt.Load()
							}
						},
					},
				},
			},
		},
	}.Run(ui.mw)
}

// Tip 显示提示信息
// category 消息类型: notify 任务栏提示消息, info 信息提示框, warning 警告信息, error 错误提示
// level 消息级别：0 不记录日志 1 严重错误 2 常规错误 3 信息提示 4 调试信息
// msg 消息内容：如果只有一个值表示消息内容，如果有两个值第一个是标题第二个是内容体
func (ui *UIMainWindow) Tip(category string, level int, msg ...string) {
	if ui.opt.Debug > 0 && level <= ui.opt.Debug {
		if nil != ui.log {
			var buf = new(bytes.Buffer)
			buf.WriteString(time.Now().Format("2006-01-02 15:04:05"))
			buf.WriteString(" [")
			buf.WriteString(category)
			buf.WriteString("] ")

			for _, v := range msg {
				buf.WriteString(v)
			}

			buf.WriteString("\r\n")

			ui.log.Write(buf.Bytes())
		}

		var title, message string
		if len(msg) > 1 {
			title = msg[0]
			message = msg[1]
		} else if len(msg) > 0 {
			title = "数据通天下 - 快捷报关数据传输助手"
			message = msg[0]
		} else {
			return
		}

		switch category {
		case "notify":
			if 1 == level || 2 == level {
				ui.ni.ShowError(title, message)
			} else if 3 == level {
				ui.ni.ShowInfo(title, message)
			}
		case "info":
			if level < 4 {
				walk.MsgBox(ui.mw, title, message, walk.MsgBoxIconInformation)
			}
		case "warning":
			if level < 4 {
				walk.MsgBox(ui.mw, title, message, walk.MsgBoxIconWarning)
			}
		case "error":
			if level < 4 {
				walk.MsgBox(ui.mw, title, message, walk.MsgBoxIconError)
			}
		default:
		}
	}
}

// SetCounter 更新计数器
func (ui *UIMainWindow) SetCounter(c *Counter) {
	ui.sbiDown.SetText("暂存：" + strconv.FormatUint(c.Download, 10))
	ui.sbiUpload.SetText("审核：" + strconv.FormatUint(c.Upload, 10))
	ui.sbiError.SetText("出错：" + strconv.FormatUint(c.Error, 10))
}

// Run 运行程序
func (ui *UIMainWindow) Run() int {
	var ret = -1
	var err = ui.Show()
	if nil == err {
		ret = ui.mw.Run()
	}

	ui.Clean()

	return ret
}

// Clean 清理资源
func (ui *UIMainWindow) Clean() {
	if nil != ui.ni {
		ui.ni.Dispose()
	}
	if nil != ui.log {
		ui.log.Close()
	}
}
