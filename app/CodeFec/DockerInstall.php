<?php

declare(strict_types=1);
/**
 * This file is part of zhuchunshu.
 * @link     https://github.com/zhuchunshu
 * @document https://github.com/zhuchunshu/SForum
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/SForum/blob/master/LICENSE
 */
namespace App\CodeFec;

use App\Command\ServerDocker;
use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\Models\Topic;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Schema\Schema;
use Hyperf\DbConnection\Db;
use Hyperf\Utils\Str;
use PDOException;
use Swoole\Coroutine\System;
use Symfony\Component\Console\Application;
use Symfony\Component\Console\Input\ArrayInput;
use Symfony\Component\Console\Output\NullOutput;
use Symfony\Component\Console\Output\OutputInterface;

/**
 * 启动服务
 */
class DockerInstall
{
    public OutputInterface $output;

    public ServerDocker $command;

    public function __construct(OutputInterface $output, ServerDocker $command)
    {
        $this->output = $output;
        $this->command = $command;
    }

    // 开始运行
    public function run()
    {
        copy(BASE_PATH . '/.env.example', BASE_PATH . '/.env');
        $dbname = env('DB_DATABASE');
        $host = env('DB_HOST');
        $port = env('DB_PORT');
        $username = env('DB_USERNAME');
        $password = env('DB_PASSWORD');

        $dsn = 'mysql:dbname=' . $dbname . ';host=' . $host . ';port=' . $port;
        try {
            $pdo = new \PDO($dsn, $username, $password);
            $this->command->info('开始安装!');
            // 数据库连接
            $this->db();
            // redis连接
            $this->redis();
            // 数据迁移
            $this->migrate();
            // 升级v2
            $this->update_v2();
        } catch (PDOException $e) {
            $this->command->error('数据库连接失败! 3秒后重试');
            $this->command->error($e->getMessage());
            sleep(3);
            $this->run();
        }
    }

    // 配置数据库信息
    public function db()
    {
        $dbname = env('DB_DATABASE');
        $host = env('DB_HOST');
        $port = env('DB_PORT');
        $username = env('DB_USERNAME');
        $password = env('DB_PASSWORD');

        $dsn = 'mysql:dbname=' . $dbname . ';host=' . $host . ';port=' . $port;
        try {
            $pdo = new \PDO($dsn, $username, $password);
            $this->command->info('数据库连接成功!');
        } catch (PDOException $e) {
            $this->command->error('数据库连接失败! 无法进行安装');
            $this->command->error($e->getMessage());
            return;
        }

        // 数据库连接成功
        modifyEnv([
            'APP_KEY' => Str::random(32),
        ]);
    }

    //配置redis信息
    public function redis()
    {
        $host = env('REDIS_HOST');
        $port = env('REDIS_PORT');
        try {
            redis()->connect($host, (int) $port);
            $this->command->info('redis连接成功!');
        } catch (\RedisException $e) {
            $this->command->error('redis连接失败! 无法进行安装');
            $this->command->error($e->getMessage());
            return;
        }
    }

    // 数据库迁移
    public function migrate()
    {
        \Swoole\Coroutine\System::exec('cp ./app/Plugins/*/src/mig*/* ./mi*');
        $command = 'migrate';

        $params = ['command' => $command];

        $input = new ArrayInput($params);
        $output = new NullOutput();

        $container = \Hyperf\Utils\ApplicationContext::getContainer();

        /** @var Application $application */
        $application = $container->get(\Hyperf\Contract\ApplicationInterface::class);
        $application->setAutoExit(false);

        $exitCode = $application->run($input, $output);

        $this->command->info('数据库迁移成功!');
    }

    // 升级v2
    public function update_v2()
    {
        // V2.0 对topic表的更改
        $topics = DB::table('topic')->where('post_id', '=', null)->get(['id', 'content', 'user_agent', 'user_ip', 'user_id', 'created_at', 'updated_at']);
        foreach ($topics as $data) {
            $post = Post::query()->create([
                'topic_id' => $data->id,
                'user_id' => $data->user_id,
                'content' => $data->content,
                'user_agent' => $data->user_agent,
                'user_ip' => $data->user_ip,
                'created_at' => $data->created_at,
                'updated_at' => $data->updated_at,
            ]);
            Topic::query()->where('id', $data->id)->update(['post_id' => $post['id']]);
        }

        // v2.0对topic_comment表的更改
        $comments = Db::table('topic_comment')->where('post_id', '=', null)->get(['id', 'user_id', 'content', 'user_agent', 'user_ip', 'created_at', 'updated_at']);
        foreach ($comments as $data) {
            $post = Post::query()->create([
                'comment_id' => $data->id,
                'user_id' => $data->user_id,
                'content' => $data->content,
                'user_agent' => $data->user_agent,
                'user_ip' => $data->user_ip,
                'created_at' => $data->created_at,
                'updated_at' => $data->updated_at,
            ]);
            TopicComment::query()->where('id', $data->id)->update(['post_id' => $post['id']]);
        }
        // v2.0清理数据库字段
        Schema::table('topic', function (Blueprint $table) {
            // 删除 多余字段
            $table->dropColumn(['content', 'markdown', 'user_agent', 'user_ip', 'like', '_token']);
        });
        Schema::table('topic_comment', function (Blueprint $table) {
            // 删除 多余字段
            $table->dropColumn(['content', 'markdown', 'user_agent', 'user_ip', 'likes']);
        });

        // gen:auth-env
        $command = 'gen:auth-env';

        $params = ['command' => $command];

        $input = new ArrayInput($params);
        $output = new NullOutput();

        $container = \Hyperf\Utils\ApplicationContext::getContainer();

        /** @var Application $application */
        $application = $container->get(\Hyperf\Contract\ApplicationInterface::class);
        $application->setAutoExit(false);

        $exitCode = $application->run($input, $output);
        $this->command->info('所有配置已完成!');
    }
}
