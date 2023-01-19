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

use App\Command\StartCommand;
use App\Plugins\Comment\src\Model\TopicComment;
use App\Plugins\Core\src\Models\Post;
use App\Plugins\Topic\src\Models\Topic;
use Hyperf\Database\Schema\Blueprint;
use Hyperf\Database\Schema\Schema;
use Hyperf\DbConnection\Db;
use Hyperf\Utils\Str;
use PDOException;
use Symfony\Component\Console\Application;
use Symfony\Component\Console\Input\ArrayInput;
use Symfony\Component\Console\Output\NullOutput;
use Symfony\Component\Console\Output\OutputInterface;

/**
 * 启动服务
 */
class Install
{
    public OutputInterface $output;

    public StartCommand $command;

    public function __construct(OutputInterface $output, StartCommand $command)
    {
        $this->output = $output;
        $this->command = $command;
    }

    // 开始运行
    public function run()
    {
        // 初始化
        if ($this->init() === false) {
            $this->command->info('初始化成功! 请重新运行此命令');
            return;
        }
        if ($this->getStep() >= 5) {
            $this->command->info('请打开网站进行最后的安装操作');
            return;
        }
        $method = 'step_' . $this->getStep();
        $this->{$method}();
    }

    // 初始化
    public function init()
    {
        if (! file_exists(BASE_PATH . '/.env')) {
            copy(BASE_PATH . '/.env.example', BASE_PATH . '/.env');
            return false;
        }
        return true;
    }

    // 步数加1
    public function addStep()
    {
        file_put_contents(BASE_PATH . '/app/CodeFec/storage/install.step.lock', $this->getStep() + 1);
    }

    public function getTips()
    {
        $tips = match ($this->getStep()) {
            1 => '配置数据库信息',
            2 => '配置redis信息',
            3 => '重启服务',
            4 => '配置服务端口',
            5 => '创建管理员账号!',
            6 => '安装完成!'
        };
        return $tips;
    }

    public function getStep()
    {
        if (! is_dir(BASE_PATH . '/app/CodeFec/storage')) {
            \Swoole\Coroutine\System::exec('mkdir ' . BASE_PATH . '/app/CodeFec/storage');
        }
        // 创建文件
        if (! file_exists(BASE_PATH . '/app/CodeFec/storage/install.step.lock')) {
            file_put_contents(BASE_PATH . '/app/CodeFec/storage/install.step.lock', 1);
        }
        if (! @file_get_contents(BASE_PATH . '/app/CodeFec/storage/install.step.lock')) {
            $step = 1;
        } else {
            $step = (int) file_get_contents(BASE_PATH . '/app/CodeFec/storage/install.step.lock');
        }
        return $step;
    }

    // 配置数据库信息
    public function step_1()
    {
        $this->command->info($this->getTips());

        // 数据库地址
        $host = $this->command->ask('数据库地址', env('DB_HOST'));
        $this->command->line($host);
        // 端口
        $port = $this->command->ask('数据库端口', env('DB_PORT'));
        $this->command->line($port);
        // 数据库用户名
        $username = $this->command->ask('数据库用户名', env('DB_USERNAME'));
        $this->command->line($username);
        // 数据库名
        $dbname = $this->command->ask('数据库名', env('DB_DATABASE'));
        $this->command->line($dbname);
        // 数据库密码
        $password = $this->command->ask('数据库密码', env('DB_PASSWORD'));
        $this->command->line($password);

        $HASH_DRIVER = $this->command->ask('密码加密驱动 [ bcrypt | md5 | md5t | argon2i | argon2id ]', 'bcrypt');
        $this->command->line($HASH_DRIVER);

        $dsn = 'mysql:dbname=' . $dbname . ';host=' . $host . ';port=' . $port;
        try {
            $pdo = new \PDO($dsn, $username, $password);
        } catch (PDOException $e) {
            $this->command->error('数据库连接失败! 无法进行安装');
            $this->command->error($e->getMessage());
            return;
        }

        // 数据库连接成功
        modifyEnv([
            'APP_KEY' => Str::random(32),
            'DB_HOST' => $host,
            'DB_PORT' => $port,
            'DB_USERNAME' => $username,
            'DB_PASSWORD' => $password,
            'DB_DATABASE' => $dbname,
            'HASH_DRIVER' => $HASH_DRIVER,
        ]);
        $this->addStep();
        $this->command->info('数据库信息配置成功!');
        $this->command->info("\n请重新运行此命令!");
    }

    //配置redis信息
    public function step_2()
    {
        $this->command->info($this->getTips());
        // redis 地址
        $host = $this->command->ask('redis 主机地址', env('REDIS_HOST'));
        $this->command->line($host);
        $port = $this->command->ask('redis 端口', env('REDIS_PORT'));
        $this->command->line($port);
        try {
            redis()->connect($host, (int) $port);
        } catch (\RedisException $e) {
            $this->command->error('redis连接失败! 无法进行安装');
            $this->command->error($e->getMessage());
            return;
        }

        // redis连接成功!
        modifyEnv([
            'REDIS_HOST' => $host,
            'REDIS_PORT' => $port,
        ]);
        $this->addStep();
        $this->command->info('redis信息配置成功!');
        $this->command->info("\n请重新运行此命令!");
    }

    // 数据库迁移
    public function step_3()
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


        // 数据填充
        $command = 'db:seed';

        $params = ['command' => $command];

        $input = new ArrayInput($params);
        $output = new NullOutput();

        $container = \Hyperf\Utils\ApplicationContext::getContainer();

        /** @var Application $application */
        $application = $container->get(\Hyperf\Contract\ApplicationInterface::class);
        $application->setAutoExit(false);

        $exitCode = $application->run($input, $output);

        // 下一步
        $this->addStep();

        $this->command->info('数据库填充成功!');
        $this->command->info("\n请重新运行此命令!");
    }

    // 配置端口
    public function step_4()
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

        $this->command->info($this->getTips());
        $web = $this->command->ask('WEB服务端口', env('SERVER_WEB_PORT'));
        $this->command->line($web);
        modifyEnv([
            'SERVER_WEB_PORT' => $web,
            'SERVER_WS_PORT' => 9502,
        ]);

        // gen auth-env

        // 数据填充
        $command = 'gen:auth-env';

        $params = ['command' => $command];

        $input = new ArrayInput($params);
        $output = new NullOutput();

        $container = \Hyperf\Utils\ApplicationContext::getContainer();

        /** @var Application $application */
        $application = $container->get(\Hyperf\Contract\ApplicationInterface::class);
        $application->setAutoExit(false);

        $exitCode = $application->run($input, $output);


        $this->addStep();
        $this->command->info('配置成功!');
        $this->command->info("\nWEB服务端口:" . $web);
        $this->command->info("\n请根据文档将对应端口进行反向代理!");
        $this->command->info("\n反代完成后打开网站进行最后一步安装!");
        $this->command->info("\n请重新运行此命令启动服务!");
        $this->command->info("\n请重新运行此命令启动服务!");
        $this->command->info("\n请重新运行此命令启动服务!");
    }
}
