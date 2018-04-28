# 说明
基于 walk(github.com/lxn/walk) 的 Windows 图形界面客户端工具，实现从网站端下载单一窗口报文到单一窗口导入客户端文件夹并把回执上传到服务器端。

# 程序编译
~~~ shell
# 编译资源文件
windres -o swa.syso ./res/swa.rc 

# 编译当前系统对应的构架版本
go build -ldflags="-H windowsgui -linkmode internal  -w" 

# 编译 32 位版
GOARCH=386 go build -ldflags="-H windowsgui -linkmode internal  -w" 
~~~ 