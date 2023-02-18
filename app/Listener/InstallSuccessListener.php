<?php

namespace App\Listener;

use App\Event\SuccessInstallEvent;
use Hyperf\Database\Schema\Schema;
use Hyperf\DB\DB;
use Hyperf\Event\Annotation\Listener;
use Hyperf\Event\Contract\ListenerInterface;
use Hyperf\Utils\ApplicationContext;

/**
 * 安装成功
 */
#[Listener]
class InstallSuccessListener implements ListenerInterface
{
    public function listen(): array
    {
        // 返回一个该监听器要监听的事件数组，可以同时监听多个事件
        return [
            SuccessInstallEvent::class,
        ];
    }

    /**
     * @param SuccessInstallEvent $event
     */
    public function process(object $event): void
    {
        // 转换user_options数据类型

        $db = ApplicationContext::getContainer()->get(DB::class);
        if (Schema::getColumnType('users_options', 'money') !== 'integer') {
            $db->query('ALTER TABLE users_options MODIFY money INT;');
        }
        // 积分
        if (Schema::getColumnType('users_options', 'credits') !== 'integer') {
            $db->query('ALTER TABLE users_options MODIFY credits INT;');
        }
        // 金币
        if (Schema::getColumnType('users_options', 'golds') !== 'integer') {
            $db->query('ALTER TABLE users_options MODIFY golds INT;');
        }
        // 经验
        if (Schema::getColumnType('users_options', 'exp') !== 'integer') {
            $db->query('ALTER TABLE users_options MODIFY exp INT;');
        }
    }
}