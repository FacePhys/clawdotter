set -e
git pull
npm i
openclaw plugins install -l .
openclaw gateway restart
echo "已经打开日志跟踪来调试，如果你嫌烦可以^C关掉不影响进程"
openclaw logs --follow

