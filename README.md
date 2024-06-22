# webook

该网站原本是为了我在极客时间上的课程而服务的，但是现在我决定将它重构为一个专注于提供面试方案的网站。

这一次我会尝试一些新的写法，所以和你们在我课程上讲述的内容会有一些出入。

但是指导思想是一致的。

此外，作为一个小应用，所以还有一些和课程内容有出入，因为我希望进一步提高研发效率。具体的点：
- 在初始阶段，摒弃定义一些没有多个实现的接口，确保研发效率。在需要的时候，再引入接口；
- 受制于 wire 的功能有限，所以一些和 ioc 有关的代码，奇丑无比

## 一点点设计原则
- 药医不死病，佛渡有缘人
- 不需要考虑攻击者的用户体验

## 缓存的 key 设计
基本上遵循了：`webook:$module:xxxxxxx` 的形式。即第一段是 webook，代表本体；第二段是 webook 内部的 module，代表模块。后面的就是 key，可以进一步细分。

## HTTP 响应码
- 大多数情况下是 200
- 未登录是 401
- 没有权限是 403

> 这里比较蛋疼的是 401 和 403 的语义。所以我也没什么好纠结的，只是做一个简单的区分

## 商品SPU类别说明
- category0表示SPU顶级类别,可选值有product表示商品,code表示兑换码
- category1表示SPU次级类别,可选值有member/project等

两者组合语义如下:
- category0=product, category1=member 表示会员商品
- category0=code, category1=project 表示项目兑换码

## 营销模块说明

1. 营销模块中兑换码相关业务为了保持独立,没有采用商品模块中的业务术语. 比如兑换码本事就蕴含了商品SPU类别category0=code这个含义,兑换码的type的值也就是商品SPU类别category1的值.

## 错误码
- user - 01
- question - 02
- cos - 03
- product - 04
- case - 05
- order - 06
- skill - 07
- label - 08
- feedback -09
- credit - 10
- project - 11
- marketing - 12
