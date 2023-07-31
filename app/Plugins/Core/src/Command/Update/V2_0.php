<?php

declare (strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\Plugins\Core\src\Command\Update;

use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\Models\Topic;
use Hyperf\Command\Annotation\Command;
use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Schema\Schema;
use Hyperf\DbConnection\Db;
use Psr\Container\ContainerInterface;
use Swoole\Coroutine\System;
#[Command]
class V2_0 extends HyperfCommand
{
    /**
     * @var ContainerInterface
     */
    protected $container;
    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;
        parent::__construct('update:2_0');
    }
    public function configure()
    {
        parent::configure();
        $this->setDescription('SForum 2.0 升级迁移命令');
    }
    public function handle()
    {
        if (file_exists(BASE_PATH . '/app/CodeFec/storage/update/2_0.lock')) {
            $this->error('你已经运行过此升级命令了,安全起见,禁止再次运行此命令');
            $this->line('如有问题,请到论坛反馈: https://runpod.cn');
            return;
        }
        $this->error('执行前一定要备份数据库 !!!');
        $this->error('Make sure to backup the database before executing !!!');
        $ask_backup = $this->ask('如果你已经备份数据库,请输入yes继续执行,否则请退出程序');
        if ($ask_backup !== 'yes') {
            $this->error('退出迁移');
            exit;
        }
        $this->info('开始迁移topic表');
        $this->topic();
        $this->info('开始迁移topic_comment表');
        $this->topic_comment();
        $this->info('迁移完毕,接下来开始清理旧数据');
        if ($this->ask('这是一个高危操作,请确保你已经备份数据,确认要清理旧数据吗?(yes/no)') === 'yes') {
            $this->info('开始清理旧数据');
            $this->clean_database();
            $this->info('清理完毕');
        }
        if (!is_dir(BASE_PATH . '/app/CodeFec/storage/update')) {
            System::exec('mkdir -p ' . BASE_PATH . '/app/CodeFec/storage/update');
        }
        file_put_contents(BASE_PATH . '/app/CodeFec/storage/update/2_0.lock', time());
        $this->info('Successfully!');
    }
    private function topic()
    {
        $topics = DB::table('topic')->where('post_id', '=', null)->get(['id', 'content', 'user_agent', 'user_ip', 'user_id', 'created_at', 'updated_at']);
        foreach ($topics as $data) {
            $post = Post::query()->create(['topic_id' => $data->id, 'user_id' => $data->user_id, 'content' => $data->content, 'user_agent' => $data->user_agent, 'user_ip' => $data->user_ip, 'created_at' => $data->created_at, 'updated_at' => $data->updated_at]);
            Topic::query()->where('id', $data->id)->update(['post_id' => $post['id']]);
        }
    }
    private function topic_comment()
    {
        $comments = Db::table('topic_comment')->where('post_id', '=', null)->get(['id', 'user_id', 'content', 'user_agent', 'user_ip', 'created_at', 'updated_at']);
        foreach ($comments as $data) {
            $post = Post::query()->create(['comment_id' => $data->id, 'user_id' => $data->user_id, 'content' => $data->content, 'user_agent' => $data->user_agent, 'user_ip' => $data->user_ip, 'created_at' => $data->created_at, 'updated_at' => $data->updated_at]);
            TopicComment::query()->where('id', $data->id)->update(['post_id' => $post['id']]);
        }
    }
    private function clean_database()
    {
        $this->info('清理topic表');
        Schema::table('topic', function (Blueprint $table) {
            // 删除 多余字段
            $table->dropColumn(['content', 'user_agent', 'user_ip', 'like', '_token']);
        });
        $this->info('清理topic_cpmment表');
        Schema::table('topic_comment', function (Blueprint $table) {
            // 删除 多余字段
            $table->dropColumn(['content', 'user_agent', 'user_ip', 'likes']);
        });
    }
}