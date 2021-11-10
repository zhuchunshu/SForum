<?php

namespace App\Plugins\User\src\Lib;

use App\Plugins\User\src\Models\UsersAuth;

class UserAuth
{
    public function create(int $user_id,string $token): void
    {
        if(UsersAuth::query()->where('user_id',$user_id)->exists()){
            UsersAuth::query()->where('user_id',$user_id)->delete();
        }
        UsersAuth::query()->create([
            'user_id' => $user_id,
            'token' => $token,
        ]);
    }

    public function destroy(int $user_id): void{
        if(UsersAuth::query()->where('user_id',$user_id)->exists()){
            UsersAuth::query()->where('user_id',$user_id)->delete();
        }
    }

    public function destroy_token(string $token): void{
        if(UsersAuth::query()->where('token',$token)->exists()){
            UsersAuth::query()->where('token',$token)->delete();
        }
    }
}