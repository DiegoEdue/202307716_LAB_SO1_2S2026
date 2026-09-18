#!/bin/bash

echo "-----Vuelta 1-----"
echo "lsmod | grep continfo_pr2_so1_202307716"
lsmod | grep continfo_pr2_so1_202307716
echo "------------------"
echo "cat /proc/continfo_pr2_so1_202307716"
cat /proc/continfo_pr2_so1_202307716
echo "-----Vuelta 1 - FINALIZADA-----"

echo "------Vuelta 2-----"
echo "ps aux | grep proyecto2-daemon"
ps aux | grep proyecto2-daemon
echo "------------------"
echo "crontab -l"
sudo crontab -l
echo "------------------"
echo "docker ps"
docker ps
echo "------------------"
echo "redis-cli -h localhost -p 6379 KEYS '*'"
redis-cli -h localhost -p 6379 KEYS '*'
echo "------------------"
echo "redis-cli -h localhost -p 6379 GET ram:used"
redis-cli -h localhost -p 6379 GET ram:used
echo "------------------"
echo "redis-cli -h localhost -p 6379 GET ram:total"
redis-cli -h localhost -p 6379 GET ram:total
echo "------------------"
echo "redis-cli -h localhost -p 6379 GET containers:deleted:count"
redis-cli -h localhost -p 6379 GET containers:deleted:count
echo "-----Vuelta 2 - FINALIZADA-----"