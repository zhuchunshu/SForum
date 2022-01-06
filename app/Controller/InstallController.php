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

namespace App\Controller;

use App\Model\AdminUser;
use Hyperf\HttpServer\Annotation\Controller;
use Hyperf\HttpServer\Annotation\GetMapping;
use Hyperf\HttpServer\Annotation\PostMapping;
use Hyperf\View\RenderInterface;
use HyperfExt\Hashing\Hash;
use Symfony\Component\Console\Application;
use Symfony\Component\Console\Input\ArrayInput;
use Symfony\Component\Console\Output\NullOutput;

/**
 * Class InstallController
 * @Controller
 * @package App\Controller
 */
class InstallController extends AbstractController
{
    #[GetMapping(path: "/install")]
    public function install(): \Psr\Http\Message\ResponseInterface
    {
        return view("core.install");
    }

    #[PostMapping(path: "/install")]
    public function post(): \Psr\Http\Message\ResponseInterface
    {
        $install_step = (int)cache()->get("install", 1);
        if ((request()->input('reduce', false) == "true") && $install_step !== 1) {
            cache()->set("install", $install_step - 1);
            return view("core.install." . cache()->get("install", 1));
        }

        // 如果点击下一步
        if (request()->input('next', false) == "true") {
            switch ($install_step) {
                case 1:
                    $this->post_step1();
                    cache()->set("install", $install_step + 1);
                    break;
                case 5:
                case 2:
                case 7:
                    cache()->set("install", $install_step + 1);
                    break;
                case 3:
                    $this->post_step2();
                    cache()->set("install", $install_step + 1);
                    break;
                case 4:
                    $this->post_step3();
                    cache()->set("install", $install_step + 1);
                    break;
                case 6:
                    $this->post_step4();
                    cache()->set("install", $install_step + 1);
                    break;
                case 8:
                    $this->post_step5();
                    cache()->set("install", $install_step + 1);
                    break;
                case 9:
                    $this->post_step6();
                    cache()->set("install", $install_step + 1);
                    break;
            }
        }

        return view("core.install." . cache()->get("install", 1));
    }

    public function post_step2(): void
    {
        $web_name = request()->input("name");
        $web_domain = request()->input("domain");
        $web_ssl = 'false';
        if (request()->input("ssl") === "https") {
            $web_ssl = 'true';
        }
        modifyEnv([
            'APP_NAME' => $web_name,
            'APP_DOMAIN' => $web_domain,
            'APP_SSL' => $web_ssl
        ]);
    }

    public function post_step1(): void
    {
        if (!file_exists(BASE_PATH . "/.env")) {
            copy(BASE_PATH . "/.env.example", BASE_PATH . "/.env");
        }
    }

    public function post_step3(): void
    {
        modifyEnv([
            'DB_HOST' => request()->input("DB_HOST"),
            'DB_DATABASE' => request()->input("DB_DATABASE"),
            'DB_USERNAME' => request()->input("DB_USERNAME"),
            'DB_PASSWORD' => request()->input("DB_PASSWORD"),
        ]);
        //file_put_contents(BASE_PATH."/app/CodeFec/storage/install-step.txt",date("Y-m-d H:i:s"));
    }

    public function post_step4(): void
    {
        modifyEnv([
            'REDIS_PORT' => request()->input("REDIS_PORT"),
            'REDIS_AUTH' => request()->input("REDIS_AUTH"),
            'REDIS_HOST' => request()->input("REDIS_HOST"),
        ]);
        $command = 'migrate';

        $params = ["command" => $command, "--force"];
        $input = new ArrayInput($params);
        $output = new NullOutput();

        $container = \Hyperf\Utils\ApplicationContext::getContainer();

        $application = $container->get(\Hyperf\Contract\ApplicationInterface::class);
        $application->setAutoExit(false);

        $exitCode = $application->run($input, $output);

        $command = 'CodeFec:migrate';

        $params = ["command" => $command];
        $input = new ArrayInput($params);
        $output = new NullOutput();

        $container = \Hyperf\Utils\ApplicationContext::getContainer();

        $application = $container->get(\Hyperf\Contract\ApplicationInterface::class);
        $application->setAutoExit(false);

        $exitCode = $application->run($input, $output);
    }

    public function post_step5(): void
    {
        AdminUser::query()->create([
            'email' => request()->input("email"),
            'username' => request()->input("username"),
            'password' => Hash::make(request()->input("password")),
        ]);
    }

    public function post_step6(): void
    {
        if (!is_dir(BASE_PATH . "/app/CodeFec/storage")) {
             \Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/app/CodeFec/storage");
        }
        if (!file_exists(BASE_PATH . "/app/CodeFec/storage/install.lock")) {
            file_put_contents(BASE_PATH . "/app/CodeFec/storage/install.lock", date("Y-m-d H:i:s"));
        }
    }
}
