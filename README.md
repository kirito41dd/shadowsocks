# shadowsockets

科学爱国





## ss-local

构建

```shell
git clone https://github.com/zshorz/shadowsockets.git
cd shadowsockets/ss-local
go build -o ss-local.exe .
./ss-local.exe -c ../config.json
```

参数说明

```shell
# 每个参数都有默认值， 如果找不到配置文件，在运行目录生成名为 config.json 的配置文件
-version 		# 打印版本后退出
-c filename 	# 指定配置文件
-s server		# 指定服务器
-p serverPort	# 指定服务器端口
-k passwd		# 服务器密码
-b address 		# 本地地址
-l port			# 本地端口
-m method		# 加密方法
-t timeout		# 超时时间
-d				# debug模式
-w 				# 命令行输入的配置写入配置文件
-u 				# 通过URI导入
```

例子

``` shell	
./ss-local.exe -c ../config.json -d -w -l 1080
```

