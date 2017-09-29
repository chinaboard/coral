##其它服务接入方式

###准备
* /configure/下创建"platform_服务.go"(可忽略)
* /configure/option.go中添加对应的Parse方法
* 修改cc中的ConfigType配置节

###Demo
* /configure/option.go添加ParseDemo方法，调用platform_demo.go的方法
```
func (op optionParser) ParseDemo() {
	AllConfig.Content = []string{
		"proxy = socks5://127.0.0.1:1080",
		"listen = http://127.0.0.1:5438"}
}
```
* go build 编译
* ConfigType = Demo
* 运行查看效果