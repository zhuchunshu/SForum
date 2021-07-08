<?php

declare(strict_types=1);
/**
 * CodeFec - Hyperf
 *
 * @link     https://github.com/zhuchunshu
 * @document https://codefec.com
 * @contact  laravel@88.com
 * @license  https://github.com/zhuchunshu/CodeFecHF/blob/master/LICENSE
 */
namespace App\Command\CodeFec;

use App\Model\AdminUser;
use Hyperf\Command\Annotation\Command;
use Hyperf\Command\Command as HyperfCommand;
use HyperfExt\Hashing\Hash;
use Psr\Container\ContainerInterface;

/**
 * @Command
 */
class InitCommand extends HyperfCommand
{
    /**
     * @var ContainerInterface
     */
    protected $container;

    public function __construct(ContainerInterface $container)
    {
        $this->container = $container;

        parent::__construct('CodeFec:Init');
    }

    public function configure()
    {
        parent::configure();
        $this->setDescription('CodeFec 初始化');
    }

    public function handle()
    {
        if (! AdminUser::count()) {
            $email = $this->ask('管理员邮箱', '123456@qq.com');
            $username = $this->ask('管理员用户名', 'admin');
            $password = $this->ask('管理员密码', '123456');
            AdminUser::query()->create([
                'email' => $email,
                'username' => $username,
                'password' => Hash::make($password),
            ]);
            $this->info('管理员账号创建成功!');
        } else {
            $this->line('已有管理员账号,无需重复创建', 'error');
        }
    }
}
