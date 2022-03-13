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
	    if (!file_exists(BASE_PATH . "/.env")) {
		    copy(BASE_PATH . "/.env.example", BASE_PATH . "/.env");
	    }
		
	    if (!is_dir(BASE_PATH . "/app/CodeFec/storage")) {
		    \Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/app/CodeFec/storage");
	    }
	    file_put_contents(BASE_PATH."/app/CodeFec/storage/install.reload.lock",time());
        return view("core.install");
    }
	
	#[PostMapping(path: "/install/env")]
	public function env(): array
	{
		if (!file_exists(BASE_PATH . "/.env")) {
			copy(BASE_PATH . "/.env.example", BASE_PATH . "/.env");
		}
		
		if (!is_dir(BASE_PATH . "/app/CodeFec/storage")) {
			\Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/app/CodeFec/storage");
		}
		$result = [];
		$content = file_get_contents(BASE_PATH."/.env");
		foreach (explode("\n",$content) as $value) {
			if($value && is_string($value)){
				$arr = explode("=",$value);
				$result[$arr[0]]=$arr[1];
			}
		}
		return $result;
	}
	
	// 上一步
	#[PostMapping(path:"/install/previous")]
	public function previous(){
		if(!file_get_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock")){
			$step = 1;
		}else{
			$step = (int)core_default(@file_get_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock"),1);
		}
		if($step<0){
			return Json_Api(403,false,['msg' => '出错啦!']);
		}
		file_put_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock",$step-1);
		return Json_Api(200, true, ['msg' => 'successfully !']);
	}
	
	// 下一步
	#[PostMapping(path: "/install/next")]
	public function next()
	{
		if(!file_get_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock")){
			$step = 1;
		}else{
			$step = (int)core_default(@file_get_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock"),1);
		}
		if($step>=5){
			return Json_Api(403,false,['msg' => '出错啦!']);
		}
		$method = "step_".$step;
		file_put_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock",$step+1);
		return $this->$method(request());
	}

	// 配置mysql
	public function step_1($request){
		if($request->input('env')){
			$env = de_stringify($request->input('env'));
			if(!is_array($env)){
				return Json_Api(403,false,['msg' => '请提交正确的数据']);
			}
			$env_arr = [];
			foreach ($env as $key=>$value){
				if($key && $value && is_string($key) && $key !== "=" && is_string($value) && $value!="="){
					$env_arr[$key] = $value;
				}
			}
			modifyEnv($env_arr);
		}
		return Json_Api(200, true, ['msg' => '数据库信息配置成功!']);
	}
	
	// 配置redis
	public function step_2($request){
		if($request->input('env')){
			$env = de_stringify($request->input('env'));
			if(!is_array($env)){
				return Json_Api(403,false,['msg' => '请提交正确的数据']);
			}
			$env_arr = [];
			foreach ($env as $key=>$value){
				if($key && $value && is_string($key) && $key !== "=" && is_string($value) && $value!="="){
					$env_arr[$key] = $value;
				}
			}
			modifyEnv($env_arr);
		}
		return Json_Api(200, true, ['msg' => 'redis信息配置成功!']);
	}
	
	// 数据迁移
	public function step_3($request){
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
		return Json_Api(200, true, ['msg' => '数据迁移成功!']);
	}
	
	// 创建管理员账号
	public function step_4($request){
		AdminUser::query()->create([
			'email' => $request->input("email"),
			'username' => $request->input("username"),
			'password' => Hash::make($request->input("password")),
		]);
		if (!file_exists(BASE_PATH . "/app/CodeFec/storage/install.lock")) {
			file_put_contents(BASE_PATH . "/app/CodeFec/storage/install.lock", date("Y-m-d H:i:s"));
		}
		return Json_Api(200, true, ['msg' => '安装成功!']);
	}
	
	
	
    #[PostMapping(path: "/install")]
    public function post(): array
    {
	    if (!is_dir(BASE_PATH . "/app/CodeFec/storage")) {
		    \Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/app/CodeFec/storage");
	    }
	    if(!file_get_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock")){
		    $install_lock = 0;
	    }else{
		    $install_lock = (int)core_default(@file_get_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock"),1);
	    }
	
		if(!file_exists(BASE_PATH."/app/CodeFec/storage/install.step.lock")){
			file_put_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock",1);
		}
	    if(!file_get_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock")){
		    $step = 1;
	    }else{
		    $step = (int)core_default(@file_get_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock"),1);
	    }
		
		$tips = match($step){
			1=>"配置数据库信息",
			2=>"配置redis信息",
			3=>"重启服务",
			4=>"创建管理员账号",
			5=>"安装完成!"
		};
		$progress = match($step){
			1=>25,
			2=>50,
			3=>75,
			4, 5 =>100
		};
        return [
			'tips' =>$tips,
			'step' => $step,
	        'progress' =>$progress,
	        'install_lock' =>$install_lock
        ];
    }
}
