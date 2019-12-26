# insane

一个 Go 编写的 http 并发测试客户端.

结果呈现一个线型图，展示每秒的成功请求数和失败数.

可以实时查看服务器负载情况.

```
Header
    KEY:Content-Type VALUE:application/x-www-form-urlencoded // 发送表单数据
    KEY:Content-Type VALUE:application/json // 发送Json数据
Body
    KEY:username  TYPE:Int  LENGTH:10  DEFAULT:insane
    KEY:password  TYPE:string  LENGTH:10  DEFAULT:
    KEY:age  TYPE:Int  LENGTH:2  DEFAULT:

    sendData:{username:"insane", password:"wI27Ayy2S2", age:24} // 不填写默认值会随机生成一个值
Cookie
        sample1:123;sample2:456;sample3:789;...
```
