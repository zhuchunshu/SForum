<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */

namespace App\Plugins\Core\src\Command\Update;

use Hyperf\Command\Annotation\Command;
use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Database\Schema\Schema;
use Hyperf\DbConnection\Db;
use Hyperf\Utils\ApplicationContext;
use Psr\Container\ContainerInterface;
use Symfony\Component\Console\Helper\ProgressBar;

/**
 * @Command
 */
#[Command]
class v238 extends HyperfCommand
{
    /**
     * @var ContainerInterface
     */
    protected $container;

    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;

        parent::__construct('update:v2.3.8');
    }

    public function configure()
    {
        parent::configure();
        $this->setDescription('升级v2.3.8必备的数据迁移命令');
    }

    public function handle()
    {
        $this->info('开始优化库中 created_at 和 updated_at字段');

        go(function () {
            $this->op_created_at_updated_at();
        });

        go(function () {
            $this->op_user_ip();
        });
    }

    private function op_created_at_updated_at()
    {
        $tables = Schema::getAllTables();
        $db = ApplicationContext::getContainer()->get(\Hyperf\DB\DB::class);
        // 需要处理的表
        $_table = [];
        foreach ($tables as $table) {
            // 表名
            $table_name = reset($table);
            if (Schema::hasColumns($table_name, ['created_at', 'updated_at'])) {
                if (Schema::getColumnType($table_name, 'created_at') !== 'integer' && Schema::getColumnType($table_name, 'updated_at') !== 'integer') {
                    $_table[] = $table_name;
                }
            }
        }
        // 获取所有需要处理表的数据数量
        $count = 0;
        foreach ($_table as $table_name) {
            $count += Schema::getConnection()->table($table_name)->count();
        }

        // creates a new progress bar (50 units)
        $progressBar = new ProgressBar($this->output, $count);

        // starts and displays the progress bar
        $progressBar->start();

        foreach ($_table as $table_name) {
            $db->query('ALTER TABLE ' . $table_name . ' MODIFY updated_at VARCHAR(255);');
            $db->query('ALTER TABLE ' . $table_name . ' MODIFY created_at VARCHAR(255);');

            foreach (Db::table($table_name)->get(['id', 'created_at', 'updated_at']) as $data) {
                $id = $data->id;
                $a = [];
                $data->created_at = $data->created_at ?: '0000-00-00 00:00:00';
                $data->updated_at = $data->updated_at ?: '0000-00-00 00:00:00';
                $a['created_at'] = strtotime($data->created_at);
                $a['updated_at'] = strtotime($data->updated_at);
                Db::table($table_name)->where('id', $id)->update($a);
                $progressBar->advance();
            }
            // 修改字段类型
            $db->query('ALTER TABLE ' . $table_name . ' MODIFY updated_at INT;');
            $db->query('ALTER TABLE ' . $table_name . ' MODIFY created_at INT;');
        }

        // ensures that the progress bar is at 100%
        $progressBar->finish();
        $this->alert('created_at 、 updated_at 字段优化完成');
    }

    private function op_user_ip()
    {
        $tables = Schema::getAllTables();
        $db = ApplicationContext::getContainer()->get(\Hyperf\DB\DB::class);
        // 需要处理的表
        $_table = [];
        foreach ($tables as $table) {
            // 表名
            $table_name = reset($table);
            if (Schema::hasColumns($table_name, ['user_ip'])) {
                if (Schema::getColumnType($table_name, 'user_ip') !== 'integer') {
                    $_table[] = $table_name;
                }
            }
        }
        // 获取所有需要处理表的数据数量
        $count = 0;
        foreach ($_table as $table_name) {
            $count += Schema::getConnection()->table($table_name)->count();
        }

        // creates a new progress bar (50 units)
        $progressBar = new ProgressBar($this->output, $count);

        // starts and displays the progress bar
        $progressBar->start();

        foreach ($_table as $table_name) {
            $db->query('ALTER TABLE ' . $table_name . ' MODIFY user_ip VARCHAR(255);');

            foreach (Db::table($table_name)->get(['id', 'user_ip']) as $data) {
                $id = $data->id;
                $a = [];
                $data->user_ip = $data->user_ip ?: '';
                $a['user_ip'] = ip2long($data->user_ip);
                Db::table($table_name)->where('id', $id)->update($a);
                $progressBar->advance();
            }
            // 修改字段类型
            $db->query('ALTER TABLE ' . $table_name . ' MODIFY user_ip INT;');
        }
        // ensures that the progress bar is at 100%
        $progressBar->finish();
        $this->alert('user_ip 字段优化完成');
    }
}
