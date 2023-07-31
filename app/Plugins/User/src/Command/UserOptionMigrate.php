<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\User\src\Command;

use App\Plugins\User\src\Models\UsersOption;
use Hyperf\Command\Annotation\Command;
use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Database\Schema\Schema;
use Hyperf\DB\DB;
use Hyperf\Utils\ApplicationContext;
use Psr\Container\ContainerInterface;
#[Command]
class UserOptionMigrate extends HyperfCommand
{
    /**
     * @var ContainerInterface
     */
    protected $container;
    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;
        parent::__construct('plugin:userOptionsMigrate');
    }
    public function configure()
    {
        parent::configure();
        $this->setDescription('Hyperf Demo Command');
    }
    public function handle()
    {
        $this->comment("\n转换用户财富数据为整型\n数值不会精确到小数点，所以用户财富中的小数点会被抹去");
        if ($this->ask('输入yes开始') !== 'yes') {
            $this->info('Bye bye');
            return;
        }
        $db = ApplicationContext::getContainer()->get(DB::class);
        // 处理数据
        foreach (UsersOption::get() as $item) {
            // 积分
            $credits = intval(trim((string) $item->credits) ?: 0);
            // 金币
            $golds = intval(trim((string) $item->golds) ?: 0);
            // 余额
            $money = intval(trim((string) $item->money) ?: 0);
            // 经验
            $exp = intval(trim((string) $item->exp) ?: 0);
            $u = UsersOption::where('id', $item->id)->update(['credits' => $credits, 'golds' => $golds, 'money' => $money, 'exp' => $exp]);
            if ($u) {
                $this->info('ID:' . $item->id . '数据转换成功');
            }
        }
        // 修改数据库字段
        // 余额
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
        $this->alert('数据转换完成');
    }
}