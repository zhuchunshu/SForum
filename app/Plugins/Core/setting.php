<?php
Itf_Setting()->add(200,
    "主题 - 全局","theme-common","plugins.Core.setting.theme.common");

Itf_Setting()->add(201,
"主题 - 页头","theme-header","plugins.Core.setting.theme.header");
Itf_Setting()->add(202,
    "主题 - 页脚","theme-footer","plugins.Core.setting.theme.footer");
Itf()->add("core_menu",1,[
    "name" => "首页",
    "url" => "/"
]);
Itf_Setting()->add(203,
    "注册设置","user-register","plugins.Core.setting.user.register");
Itf_Setting()->add(204,
    "用户设置","user-setting","plugins.Core.setting.user.core");



