Installation
============
Clone into /opt/watchandlisten and go build watchandlisten.go
Copy watchandlisten.init.d to /etc/init.d/watchandlisten

Running
-------
/etc/init.d/watchandlisten start|stop|restart
watchandlisten -conf <conf path>
watchandlisten -test
watchandlisten -run <webhook path>

Config
------

Monit
-----
Add monit script /etc/monit.d/watchandlisten
```
check process watchandlisten with pidfile /var/run/watchandlisten.pid
    start program = "/etc/init.d/watchandlisten start"
    stop program = "/etc/init.d/watchandlisten stop"
```


