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
use App\Plugins\User\src\Models\UserClass;
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
	
	
	// 下一步
	#[PostMapping(path: "/install/next")]
	public function next()
	{
		if(!file_get_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock")){
			$step = 1;
		}else{
			$step = (int)core_default(@file_get_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock"),1);
		}
		if($step>=6){
			return Json_Api(403,false,['msg' => '出错啦!']);
		}
		$method = "step_".$step;
		file_put_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock",$step+1);
		return $this->$method(request());
	}
	
	public function step_5($request){
		AdminUser::query()->create([
			'email' => $request->input("email"),
			'username' => $request->input("username"),
			'password' => Hash::make($request->input("password")),
		]);
		if (!file_exists(BASE_PATH . "/app/CodeFec/storage/install.lock")) {
			file_put_contents(BASE_PATH . "/app/CodeFec/storage/install.lock", date("Y-m-d H:i:s"));
		}
        go(function (){
            \App\Plugins\Topic\src\Models\TopicTag::query()->create([
                'name' => '默认板块',
                'color' => '#000000',
                'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-message" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <path d="M4 21v-13a3 3 0 0 1 3 -3h10a3 3 0 0 1 3 3v6a3 3 0 0 1 -3 3h-9l-4 4"></path>
   <line x1="8" y1="9" x2="16" y2="9"></line>
   <line x1="8" y1="13" x2="14" y2="13"></line>
</svg>',
                'description' => '默认板块描述',
                'type' => null,
                'font_color' => '#FFFFFF'
            ]);
            UserClass::query()->create([
                'name' => '默认用户组',
                'color' => '#206bc4',
                'icon' => '<svg xmlns="http://www.w3.org/2000/svg" class="icon icon-tabler icon-tabler-users" width="24" height="24" viewBox="0 0 24 24" stroke-width="2" stroke="currentColor" fill="none" stroke-linecap="round" stroke-linejoin="round">
   <path stroke="none" d="M0 0h24v24H0z" fill="none"></path>
   <circle cx="9" cy="7" r="4"></circle>
   <path d="M3 21v-2a4 4 0 0 1 4 -4h4a4 4 0 0 1 4 4v2"></path>
   <path d="M16 3.13a4 4 0 0 1 0 7.75"></path>
   <path d="M21 21v-2a4 4 0 0 0 -3 -3.85"></path>
</svg>',
                'quanxian' => '["comment_caina","comment_create","comment_edit","comment_remove","report_comment","report_topic","topic_create","topic_delete","topic_edit","topic_tag_create"]',
                'permission-value' => '1',
            ]);
        });
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
			4=>"配置服务端口",
			5=>"创建管理员账号!",
			6=>"安装完成!"
		};
		$progress = match($step){
			1=>25,
			2=>50,
			3=>75,
			4=>90,
			5=>100,
			6=>100
		};
        return [
			'tips' =>$tips,
			'step' => $step,
	        'progress' =>$progress,
	        'install_lock' =>$install_lock
        ];
    }
}
