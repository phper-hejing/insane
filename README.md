# insane

一个 Go 编写的 http 并发测试客户端.

使用G2Plot展示图表，实时显示服务器负载，压测请求数.

(https://i.imgur.com/gPGQuEc.png)

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
