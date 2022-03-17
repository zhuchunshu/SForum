<?php

namespace App\Plugins\User\src\Lib;

use App\Plugins\User\src\Event\SendMail;
use App\Plugins\User\src\Models\User;
use App\Plugins\User\src\Models\UsersNotice;
use App\Plugins\User\src\Models\UsersNoticed;
use Hyperf\Database\Schema\Schema;
use Hyperf\Di\Annotation\Inject;
use Psr\EventDispatcher\EventDispatcherInterface;

class UserNotice
{
	
	/**
	 * @Inject
	 * @var EventDispatcherInterface
	 */
	private $eventDispatcher;
	
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

    /**
     * 发送通知
     */
    public function send($user_id,$title,$content,$action=null): void
    {
        UsersNotice::query()->create([
            'user_id' => $user_id,
            'title' => $title,
            'content' => $content,
            'action' => $action,
            'status' => 'publish'
        ]);
	    $this->eventDispatcher->dispatch(new SendMail($user_id,$title,$action));
    }
	
	
    /**
     * 给多个用户发送通知
     */
    public function sends(array $user_ids,$title,$content,$action=null): void
    {
        foreach ($user_ids as $user_id){
            UsersNotice::query()->create([
                'user_id' => $user_id,
                'title' => $title,
                'content' => $content,
                'action' => $action,
                'status' => 'publish'
            ]);
	        $this->eventDispatcher->dispatch(new SendMail($user_id,$title,$action));
        }
    }
}