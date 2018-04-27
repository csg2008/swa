package main

// App 应用入口
type App struct {
	ui  *UIMainWindow `label:"应用界面"`
	opt *Options      `label:"配置选项"`
	exe *Execute      `label:"指令执行"`
}

// init 初始化应用程序
func (app *App) init() {
	app.opt = new(Options)
	app.exe = new(Execute)
	app.ui = new(UIMainWindow)

	app.opt.Init()
	app.exe.Init(app.opt, app.ui.Tip, app.ui.SetCounter)
	app.ui.Init(app.opt, app.exe)
}

// Run 运行应用程序
func (app *App) Run() {
	app.init()

	app.ui.Run()
}
