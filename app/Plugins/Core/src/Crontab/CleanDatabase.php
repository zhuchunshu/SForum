<?php

namespace App\Plugins\Core\src\Crontab;

use App\Plugins\Topic\src\Models\Topic;
use App\Plugins\User\src\Models\UsersNotice;
use Hyperf\Crontab\Annotation\Crontab;

/**
 * @Crontab(name="CleanDatabase", rule="0 * * * *", callback="execute", enable={CleanDatabase::class, "isEnable"}, memo="数据库垃圾清理")
 */
class CleanDatabase
{
	public function execute()
	{
		// 清理已删除的文章
		$this->topic();
		// 清理已读通知
		$this->notice();
	}
	
	public function isEnable(): bool
	{
		return true;
	}
	
	// 清理已删除的文章
	private function topic(){
		Topic::query()->where('status', 'delete')->delete();
	}
	
	// 清理已读通知
	private function notice(){
		UsersNotice::query()->where('status', 'read')->delete();
	}
}