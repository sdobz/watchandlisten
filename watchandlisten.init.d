#!/bin/bash
# chkconfig: - 99 10

CMD="/opt/watchandlisten/watchandlisten"
PIDFILE="/var/run/watchandlisten.pid"
USER="ec2-user"

. /etc/init.d/functions

start() {
  sudo -u ${USER} ${CMD} -test
  if [ -s ${PIDFILE} ]; then
    RETVAL=1
    echo -n "Already running" && warning
    echo
  else
    sudo -u ${USER} nohup ${CMD} 0<&- &>/dev/null &
    RETVAL=$?
    PID=$!
    [ $RETVAL -eq 0 ] && success || failure
    echo
    echo $PID > ${PIDFILE}
  fi
}

stop() {
  if [ -s ${PIDFILE} ]; then
    if kill -0 `cat ${PIDFILE}` 2>/dev/null; then
      kill `cat ${PIDFILE}` && success || failure
      echo
    fi
    rm ${PIDFILE}
  fi
}

case $1 in
  start)
    start
  ;;
  stop)
    stop
  ;;
  restart)
    stop
    start
  ;;
  *)  
  echo "usage: watchandlisten {start|stop|restart}" ;;
esac
exit 0
