#启动fdfs
sudo  fdfs_trackerd  /etc/fdfs/tracker.conf
sudo fdfs_storaged  /etc/fdfs/storage.conf

#启动ngnix
sudo  /usr/local/nginx/sbin/nginx

#启动redis
sudo redis-server /etc/redis/redis.conf