<?php
menu()->add(200,[
    "name" => "站点设置",
    "url" => "/admin/setting",
    "icon" => "",
    "parent_id" => 5
]);
menu()->add(201,[
    "name" => "主题设置",
    "url" => "/admin/core/setting/theme",
    "icon" => "",
    "parent_id" => 5
]);

Itf_Route()->set("/s",function(){
    return 1;
});