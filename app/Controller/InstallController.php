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
    public function install()
    {
        switch (request()->input("step")) {
            case 1:
                return view("core.install.step1");
                break;
            case 2:
                return view("core.install.step2");
                break;
            case 3:
                return view("core.install.step3");
                break;
            case 4:
                //file_put_contents(BASE_PATH."/app/CodeFec/storage/install-step.txt",date("Y-m-d H:i:s"));
                return view("core.install.step4");
                break;
            case 5:
                return view("core.install.step5");
                break;
            case 6:
                return view("core.install.step6");
                break;
            default:
                return view("core.install.step1");
                break;
        }
    }

    /**
     * @PostMapping(path="/install")
     */
    public function post()
    {
        switch (request()->input("step")) {
            case '1':
                return $this->post_step1();
                break;
            case '2':
                return $this->post_step2();
                break;
            case '3':
                return $this->post_step3();
                break;
            case '4':
                return $this->post_step4();
                break;
            case '5':
                return $this->post_step5();
                break;
            case '6':
                return $this->post_step6();
                break;

            default:
                return admin_abort(['msg' => "步骤不存在"]);
                break;
        }
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
        //file_put_contents(BASE_PATH."/app/CodeFec/storage/install-step.txt",date("Y-m-d H:i:s"));
        return response()->redirect("/install?step=3");
    }

    public function post_step1(): \Psr\Http\Message\ResponseInterface
    {
        if (!file_exists(BASE_PATH . "/.env")) {
            copy(BASE_PATH . "/.env.example", BASE_PATH . "/.env");
        }
        //file_put_contents(BASE_PATH."/app/CodeFec/storage/install-step.txt",date("Y-m-d H:i:s"));
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
        //file_put_contents(BASE_PATH."/app/CodeFec/storage/install-step.txt",date("Y-m-d H:i:s"));
        return response()->redirect("/install?step=4");
    }

    public function post_step4(): \Psr\Http\Message\ResponseInterface
    {
        modifyEnv([
            'REDIS_PORT' => request()->input("REDIS_PORT"),
            'REDIS_AUTH' => request()->input("REDIS_AUTH"),
            'REDIS_HOST' => request()->input("REDIS_HOST"),
        ]);
        file_put_contents(BASE_PATH."/app/CodeFec/storage/install-step.txt",date("Y-m-d H:i:s"));

        return response()->redirect("/install?step=5");
    }

    public function post_step5(): \Psr\Http\Message\ResponseInterface
    {
        $command = 'migrate';

        $params = ["command" => $command, "--force"];
        $input = new ArrayInput($params);
        $output = new NullOutput();

        $container = \Hyperf\Utils\ApplicationContext::getContainer();

        $application = $container->get(\Hyperf\Contract\ApplicationInterface::class);
        $application->setAutoExit(false);

        $exitCode = $application->run($input, $output);
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
        if(!file_exists(BASE_PATH."/app/CodeFec/storage/install.lock")){
            file_put_contents(BASE_PATH."/app/CodeFec/storage/install.lock",date("Y-m-d H:i:s"));
        }
        return redirect()->url("/admin")->go();
    }
}
