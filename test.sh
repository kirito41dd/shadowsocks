# 不要执行这个文件
ss-server -c config.json &
ss-local  -c config.json &

sleep 5
killall ss-server
killall ss-local