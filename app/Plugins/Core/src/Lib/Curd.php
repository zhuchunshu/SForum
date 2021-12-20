<?php

namespace App\Plugins\Core\src\Lib;

use App\Plugins\User\src\Models\UserClass;

class Curd
{
    public function GetUserClass($class_id){
        return UserClass::query()->where("id",$class_id)->first();
    }
}