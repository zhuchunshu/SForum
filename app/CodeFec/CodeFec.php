<?php
namespace App\CodeFec;

use App\Model\AdminPlugin;

class CodeFec {

    public function handle(){
        $this->menu();
        $this->header();
        $this->boot();
    }

    // 注册菜单
    public function menu(){

        require BASE_PATH."/app/CodeFec/Menu/default.php";

    }

    //创建页头内容
    public function header(){
        require BASE_PATH."/app/CodeFec/Header/default.php";
    }

    public function boot(){
        require BASE_PATH."/app/CodeFec/bootstrap.php";
    }

}