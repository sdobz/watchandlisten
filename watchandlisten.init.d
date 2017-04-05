#!/bin/bash

PIDFILE=/var/run/watchandlisten.pid
CMD="/opt/watchandlisten/watchandlisten"
USER="apache"
GROUP="apache"

. /etc/init.d/functions

start() {
  sudo -u ${USER} ${CMD} -test
  daemon --pidfile="${PIDFILE}" --user=${USER} ${CMD}
}

stop() {
  if [ -f ${PIDFILE} ]; then
    kill `cat ${PIDFILE}`
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
