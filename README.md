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

## 错误码
- user - 01
- question - 02
- cos - 03
- product - 04
- case - 05
