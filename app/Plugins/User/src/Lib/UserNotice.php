<?php

namespace App\Plugins\User\src\Lib;

use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UsersNoticed;
use Hyperf\Database\Schema\Schema;

class UserNotice
{
    public function check(string $type,int|string $user_id): bool
    {
        if(!User::query()->where("id",$user_id)->exists()){
            return false;
        }
        if(!UsersNoticed::query()->where("user_id",$user_id)->exists()){
            return false;
        }
        if(!Schema::hasColumn("users_noticed",$type)){
            return false;
        }
        if(UsersNoticed::query()->where(["user_id"=>$user_id,$type=>true])->exists()){
            return true;
        }
        return false;
    }

    public function checked(string $type,int|string $user_id): string
    {
        if($this->check($type,$user_id)){
            return "checked";
        }
        return "";
    }

    public function update($user_id,$data): void
    {
        UsersNoticed::query()->where(["user_id"=>$user_id])->delete();
        foreach ($data as $key=>$value){
            if($value==='on'){
                $value=true;
            }else{
                $value = false;
            }
            if(Schema::hasColumn("users_noticed",$key)){
                UsersNoticed::query()->where(["user_id"=>$user_id])->create([
                    $key=>$value,
                    "user_id" => $user_id
                ]);
            }
        }
    }
}