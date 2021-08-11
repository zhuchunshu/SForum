<?php

// 菜单
menu()->add(301,[
   "name" => "帖子标签",
    "url" => "#",
    "icon" => '<svg xmlns="http://www.w3.org/2000/svg" width="16" height="16" fill="currentColor" class="bi bi-tags" viewBox="0 0 16 16">
  <path d="M3 2v4.586l7 7L14.586 9l-7-7H3zM2 2a1 1 0 0 1 1-1h4.586a1 1 0 0 1 .707.293l7 7a1 1 0 0 1 0 1.414l-4.586 4.586a1 1 0 0 1-1.414 0l-7-7A1 1 0 0 1 2 6.586V2z"/>
  <path d="M5.5 5a.5.5 0 1 1 0-1 .5.5 0 0 1 0 1zm0 1a1.5 1.5 0 1 0 0-3 1.5 1.5 0 0 0 0 3zM1 7.086a1 1 0 0 0 .293.707L8.75 15.25l-.043.043a1 1 0 0 1-1.414 0l-7-7A1 1 0 0 1 0 7.586V3a1 1 0 0 1 1-1v5.086z"/>
</svg>'
]);

// 菜单
menu()->add(302,[
    "name" => "管理",
    "url" => "/admin/topic/tag",
    "icon" => '',
    "parent_id" => 301
]);

// 菜单
menu()->add(303,[
    "name" => "新增",
    "url" => "/admin/topic/tag/create",
    "icon" => '',
    "parent_id" => 301
]);



if(!function_exists("core_Str_menu_url")){
    function core_Str_menu_url(string $path): string
    {
        if($path ==="//"){
            $path = "/";
        }
        return $path;
    }
}

// 首页菜单
Itf()->add("menu",1,[
   "name" => "首页",
   "url" => "/",
    "icon" => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-home" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <polyline points="5 12 3 12 12 3 21 12 19 12"></polyline>
   <path d="M5 12v7a2 2 0 0 0 2 2h10a2 2 0 0 0 2 -2v-7"></path>
   <path d="M9 21v-6a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v6"></path>
</svg>',
]);
Itf()->add("menu",10,[
    "name" => "首页222",
    "url" => "/",
    "icon" => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-home" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <polyline points="5 12 3 12 12 3 21 12 19 12"></polyline>
   <path d="M5 12v7a2 2 0 0 0 2 2h10a2 2 0 0 0 2 -2v-7"></path>
   <path d="M9 21v-6a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v6"></path>
</svg>',
]);
// 首页菜单
Itf()->add("menu",2,[
    "name" => "首页2",
    "url" => "/",
    "icon" => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-home" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <polyline points="5 12 3 12 12 3 21 12 19 12"></polyline>
   <path d="M5 12v7a2 2 0 0 0 2 2h10a2 2 0 0 0 2 -2v-7"></path>
   <path d="M9 21v-6a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v6"></path>
</svg>',
    "parent_id" =>1
]);
Itf()->add("menu",3,[
    "name" => "首页2",
    "url" => "/",
    "icon" => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-home" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <polyline points="5 12 3 12 12 3 21 12 19 12"></polyline>
   <path d="M5 12v7a2 2 0 0 0 2 2h10a2 2 0 0 0 2 -2v-7"></path>
   <path d="M9 21v-6a2 2 0 0 1 2 -2h2a2 2 0 0 1 2 2v6"></path>
</svg>',
    "parent_id" =>1
]);

if (!function_exists("core_menu_pd")) {
    function core_menu_pd(string $id)
    {
        foreach (Itf()->get('menu') as $value) {
            if (arr_has($value, "parent_id") && "menu_" . $value['parent_id'] === (string)$id) {
                return true;
            }
        }
        return false;
    }
}

if(!function_exists("core_Itf_id")){
    function core_Itf_id($name,$id){
        return \Hyperf\Utils\Str::after($id,$name."_");
    }
}

if (!function_exists("core_menu_pdArr")) {
    function core_menu_pdArr($id): array
    {
        $arr = [];
        foreach (Itf()->get("menu") as $key => $value) {
            if (arr_has($value, "parent_id") && "menu_".$value['parent_id'] === $id) {
                $arr[$key] = $value;
            }
        }
        return $arr;
    }
}