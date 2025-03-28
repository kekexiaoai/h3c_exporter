#!/bin/bash

# 定义二进制文件和配置文件路径
BINARY_NAME="h3c_exporter"
CONFIG_FILE="config.yaml"
PID_FILE="/var/run/${BINARY_NAME}.pid"
LOG_FILE="./h3c_exporter.log"

# 启动程序
start() {
    if [ -f "$PID_FILE" ]; then
        echo "$BINARY_NAME is already running."
        return 1
    fi
    # 将程序输出重定向到日志文件
    ./$BINARY_NAME -config $CONFIG_FILE >> $LOG_FILE 2>&1 &
    echo $! > $PID_FILE
    echo "$BINARY_NAME started. Logs are stored in $LOG_FILE."
}

# 停止程序
stop() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat $PID_FILE)
        kill $PID
        rm $PID_FILE
        echo "$BINARY_NAME stopped."
    else
        echo "$BINARY_NAME is not running."
    fi
}

# 查看程序状态
status() {
    if [ -f "$PID_FILE" ]; then
        PID=$(cat $PID_FILE)
        if ps -p $PID > /dev/null 2>&1; then
            echo "$BINARY_NAME is running with PID $PID. Logs are stored in $LOG_FILE."
        else
            echo "$BINARY_NAME is not running (PID file exists but process is not found)."
            rm $PID_FILE
        fi
    else
        echo "$BINARY_NAME is not running."
    fi
}

# 根据命令行参数执行相应操作
case "$1" in
    start)
        start
        ;;
    stop)
        stop
        ;;
    status)
        status
        ;;
    *)
        echo "Usage: $0 {start|stop|status}"
        exit 1
        ;;
esac

exit 0
