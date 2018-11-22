## 简介
由于ss代理客户端提供的是socks5代理，本人需要https代理，对于单个socks代理转http代理可以使用privoxy,但是我这里需要对多个socks代理进行转换，同时为了使用简单提供统一的接口进行访问

```                                   
                                   |ss0|
                      1080         |ss1|
    client1------->|       |       |ss2|
    client2------->|proxygo|------<|ss3|
    client3------->|       |       |ss4|
                                   |ss5|
                                   |...|
```

## 配置
可以配置多个ss客户端进行，服务在1080端口监听请求，接收到请求后分发给各客户端，使用最少连接优先的策略
> 后端使用shadowsocks-go,所以所有shadowsocks-go支持的method都支持
```
{
  "Configs": [
    {
      "Server": "192.168.0.1",
      "PassWord": "test",
      "RemotePort": 2345,
      "LocalAddr": "127.0.0.1",
      "LocalPort": 1090,
      "Method": "aes-256-cfb"
    }
  ],
  "LocalPort":1080
}
```

## 启动
首先启动shadosocks客户端，启动脚本
```
nohup python boostrap_linux.py
```
