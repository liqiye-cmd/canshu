# canshu
发现页面隐藏的参数， 更好地扩大工具面

用golang 重写了https://github.com/s0md3v/Arjun
特点：
速度更快，能够以静默模式输出，子域名工具 | 测活工具 |  canshu | xss 扫描工具，以这样的方式联合其他工具。

使用方法：
go get github.com/liqiye-cmd/canshu

cat url文件 | canshu 或者 echo "url" | canshu 或者 canshu -u url
如果想要输出详细信息， 用 -v 参数
默认从当前目录params.txt文件读取参数字典，当然你也可以通过 -f 指定

