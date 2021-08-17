<?php

// 设置邮件验证白名单路径
Itf()->add("authMiddleware",1,"api*");
Itf()->add("authMiddleware",2,"admin*");
Itf()->add("authMiddleware",3,"logout");

