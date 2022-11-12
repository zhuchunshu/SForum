<?php

namespace App\Plugins\Core\src\Crontab;

use App\Plugins\Core\src\Models\PayOrder;
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
        // 清理超过一天未付款、取消、关闭的订单
        $this->order();
        // 清理admin_logger日志
        $this->admin_logger();
    }
    
    public function isEnable(): bool
    {
        return true;
    }
    
    // 清理已删除的文章
    private function topic()
    {
        Topic::query()->where('status', 'delete')->delete();
    }
    
    // 清理已读通知
    private function notice()
    {
        UsersNotice::query()->where('status', 'read')->delete();
    }

    // 清理订单
    private function order()
    {
        $data = [];
        foreach (PayOrder::query()->where('status', '待支付')
                     ->orWhere('status', '订单取消')
                     ->orWhere('status', '交易关闭')->get() as $value) {
            if (time() - strtotime($value->created_at) > 86400) {
                $data[] = $value->id;
            }
        }
        foreach ($data as $id) {
            PayOrder::where('id', $id)->delete();
        }
        //return $data;
    }

    /**
     * 清理admin_logger过期日志
     * @return void
     */
    private function admin_logger(): void
    {
        foreach (scandir(BASE_PATH."/runtime/logs/admin_logger_database") as $name) {
            if (is_dir(BASE_PATH."/runtime/logs/admin_logger_database/".$name) && $name!==(string)date('YmW') && $name!=='.' && $name!=='..') {
                deldir(BASE_PATH."/runtime/logs/admin_logger_database/".$name);
            }
        }
    }
}
