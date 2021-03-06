说明
====

这是一个分包型协议的网关框架，基于这个网关框架你可以快速搭建出一个针对具体项目需求的网关程序。

这个网关框架将系统分为以下几个角色：

**Frontend**

网关前端，就是网关服务器和应用客户端通讯的部分，它负责接收来客户端连接，并把连接分配给具体的应用服务器，分配算法由框架的使用者通过注册ClientHandshaker回调函数实现。

**Backend**

网关后端，就是应用服务器和网关前端通讯的部分，网关前后端之间通过一个Socket实例进行通讯，然后网关后端再将消息派发给不同的Session，以实现对应用服务透明的网关通讯层。

**Frontend/Backend Link**

网关链接，网关前端和网关后端之间的通讯由网关链接负责，一个网关前端可以链接到多个网关后端，也可以对同一个网关后端创建多个链接，一个网关后端可以同时接入多个网关前端，也可以同一个网关前端接入多次。
