<?php

namespace App\Plugins\Core\src\Crontab;
use App\Model\AdminOption;
use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\User\src\Models\UsersNotice;
use App\Plugins\User\src\Models\UsersPm;
use Hyperf\Crontab\Annotation\Crontab;

/**
 * 用户私信清理
 * @Crontab(name="CleanUsersPm", rule="0 * * * *", callback="execute", enable={CleanUsersPm::class, "isEnable"}, memo="用户私信清理")
 */
class CleanUsersPm
{
	public function execute()
	{
		// 消息保留时间,单位:秒
		$reserve = (int)get_options('pm_msg_reserve',7)*24*60*60;
		
		foreach(UsersPm::query()->get() as $pm){
			if(time()-strtotime($pm->created_at)>=$reserve){
				UsersPm::query()->where('id',$pm->id)->delete();
			}
		}
	}
	
	public function isEnable(): bool
	{
		return !((int)$this->get_options('pm_msg_reserve', 7) === 0);
	}
	
	private function core_default($string=null,$default=null){
		if($string){
			return $string;
		}
		return $default;
	}
	
	private function get_options($name,$default=""){
		if(!cache()->has('admin.options.'.$name)){
			cache()->set("admin.options.".$name,@AdminOption::query()->where("name",$name)->first()->value);
		}
		return $this->core_default(cache()->get("admin.options.".$name),$default);
	}
}