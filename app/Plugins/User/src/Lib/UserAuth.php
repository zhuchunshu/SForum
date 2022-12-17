<?php

namespace App\Plugins\User\src\Lib;

use App\Plugins\User\src\Models\UsersAuth;

class UserAuth
{
    public function create(int $user_id,string $token): bool
    {
	    if(UsersAuth::query()->where('user_id',$user_id)->count()>get_options('core_user_session_num', 1)){
		    UsersAuth::query()->where('user_id',$user_id)->take(1)->delete();
	    }
        UsersAuth::query()->create([
            'user_id' => $user_id,
            'token' => $token,
	        'user_ip' => get_client_ip()
        ]);
		return true;
    }

    public function destroy(string|int $user_id): void{
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