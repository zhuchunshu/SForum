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
    /**
     * @GetMapping(path="/install")
     */
    public function install(): ?\Psr\Http\Message\ResponseInterface
    {
        return match (request()->input("step")) {
            2 => view("core.install.step2"),
            3 => view("core.install.step3"),
            4 => view("core.install.step4"),
            5 => view("core.install.step5"),
            6 => view("core.install.step6"),
            default => view("core.install.step1"),
        };
    }

    /**
     * @PostMapping(path="/install")
     */
    public function post()
    {
        return match (request()->input("step")) {
            '1' => $this->post_step1(),
            '2' => $this->post_step2(),
            '3' => $this->post_step3(),
            '4' => $this->post_step4(),
            '5' => $this->post_step5(),
            '6' => $this->post_step6(),
            default => admin_abort(['msg' => "步骤不存在"]),
        };
    }

    public function post_step2(): \Psr\Http\Message\ResponseInterface
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
        return response()->redirect("/install?step=3");
    }

    public function post_step1(): \Psr\Http\Message\ResponseInterface
    {
        if (!file_exists(BASE_PATH . "/.env")) {
            copy(BASE_PATH . "/.env.example", BASE_PATH . "/.env");
        }
        return response()->redirect("/install?step=2");
    }

    public function post_step3(): \Psr\Http\Message\ResponseInterface
    {
        modifyEnv([
            'DB_HOST' => request()->input("DB_HOST"),
            'DB_DATABASE' => request()->input("DB_DATABASE"),
            'DB_USERNAME' => request()->input("DB_USERNAME"),
            'DB_PASSWORD' => request()->input("DB_PASSWORD"),
        ]);
        return response()->redirect("/install?step=4");
    }

    public function post_step4(): \Psr\Http\Message\ResponseInterface
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

        return response()->redirect("/install?step=5");
    }

    public function post_step5(): \Psr\Http\Message\ResponseInterface
    {
        AdminUser::query()->create([
            'email' => request()->input("email"),
            'username' => request()->input("username"),
            'password' => Hash::make(request()->input("password")),
        ]);
        return response()->redirect("/install?step=6");
    }

    public function post_step6(): \Psr\Http\Message\ResponseInterface
    {
        if(!is_dir(BASE_PATH."/app/CodeFec/storage")){
            exec("mkdir ".BASE_PATH."/app/CodeFec/storage");
        }
        file_put_contents(BASE_PATH."/app/CodeFec/storage/install.lock",date("Y-m-d H:i:s"));
        $params = ["command" => "CodeFec:PluginsComposerInstall"];

        $input = new ArrayInput($params);
        $output = new NullOutput();

        $container = \Hyperf\Utils\ApplicationContext::getContainer();

        /** @var Application $application */
        $application = $container->get(\Hyperf\Contract\ApplicationInterface::class);
        $application->setAutoExit(false);

        // 这种方式: 不会暴露出命令执行中的异常, 不会阻止程序返回
        $exitCode = $application->run($input, $output);
        return redirect()->url("/admin")->go();
    }
}
