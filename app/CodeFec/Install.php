<?php

namespace App\CodeFec;

use App\Command\StartCommand;
use Swoole\Coroutine\MySQL;
use Swoole\Coroutine\Redis;
use Symfony\Component\Console\Application;
use Symfony\Component\Console\Input\ArrayInput;
use Symfony\Component\Console\Output\NullOutput;
use Symfony\Component\Console\Output\OutputInterface;

/**
 * 启动服务
 */
class Install
{
	/**
	 * @var OutputInterface
	 */
	public OutputInterface $output;
	
	/**
	 * @var StartCommand
	 */
	public StartCommand $command;
	
	public function __construct(OutputInterface $output,StartCommand $command){
		$this->output = $output;
		$this->command = $command;
	}
	
	// 开始运行
	public function run(){
		if($this->getStep()>=5){
			$this->command->info("请打开网站进行最后的安装操作");
			return ;
		}
		$method = "step_".$this->getStep();
		$this->$method();
	}
	
	// 步数加1
	public function addStep(){
		file_put_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock",$this->getStep()+1);
	}
	
	public function getTips(){
		$tips = match($this->getStep()){
			1=>"配置数据库信息",
			2=>"配置redis信息",
			3=>"重启服务",
			4=>"配置服务端口",
			5=>"创建管理员账号!",
			6=>"安装完成!"
		};
		return $tips;
	}
	
	public function getStep()
	{
		if (!is_dir(BASE_PATH . "/app/CodeFec/storage")) {
			\Swoole\Coroutine\System::exec("mkdir " . BASE_PATH . "/app/CodeFec/storage");
		}
		// 创建文件
		if(!file_exists(BASE_PATH."/app/CodeFec/storage/install.step.lock")){
			file_put_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock",1);
		}
		if(!@file_get_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock")){
			$step = 1;
		}else{
			$step = (int)file_get_contents(BASE_PATH."/app/CodeFec/storage/install.step.lock");
		}
		return $step;
	}
	
	// 配置数据库信息
	public function step_1(){
		$this->command->info($this->getTips());
		
		// 数据库地址
		$host = $this->command->ask('数据库地址',env('DB_HOST'));
		$this->command->line($host);
		// 端口
		$port = $this->command->ask('数据库端口',env('DB_PORT'));
		$this->command->line($port);
		// 数据库用户名
		$username = $this->command->ask('数据库用户名',env('DB_USERNAME'));
		$this->command->line($username);
		// 数据库名
		$dbname = $this->command->ask('数据库名',env('DB_DATABASE'));
		$this->command->line($dbname);
		// 数据库密码
		$password = $this->command->ask('数据库密码',env('DB_PASSWORD'));
		$this->command->line($password);
		$swoole_mysql = new MySQL();
		$swoole_mysql->connect([
			'host'     => $host,
			'port'     => $port,
			'user'     => $username,
			'password' => $password,
			'database' => $dbname,
		]);
		if($swoole_mysql->connected===false){
			$this->command->error('数据库连接失败! 无法进行安装');
			return ;
		}
		// 数据库连接成功
		modifyEnv([
			'DB_HOST' => $host,
			'DB_PORT' => $port,
			'DB_USERNAME' => $username,
			'DB_PASSWORD' => $password,
			'DB_DATABASE' => $dbname,
		]);
		$this->addStep();
		$this->command->info('数据库信息配置成功!');
		$this->command->info("\n请重新运行此命令!");
	}
	
	//配置redis信息
	public function step_2(){
		$this->command->info($this->getTips());
		// redis 地址
		$host = $this->command->ask('redis 主机地址',env('REDIS_HOST'));
		$this->command->line($host);
		$port = $this->command->ask('redis 端口',env('REDIS_PORT'));
		$this->command->line($port);
		$redis = new Redis();
		$redis->connect($host, $port);
		if($redis->connected===false){
			$this->command->error('redis连接失败! 无法进行安装');
			return ;
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
	public function step_3(){
		$command = 'migrate';
		
		$params = ["command" => $command];
		
		$input = new ArrayInput($params);
		$output = new NullOutput();
		
		$container = \Hyperf\Utils\ApplicationContext::getContainer();
		
		/** @var Application $application */
		$application = $container->get(\Hyperf\Contract\ApplicationInterface::class);
		$application->setAutoExit(false);
		
		$exitCode = $application->run($input, $output);
		
		$this->addStep();
		$this->command->info('数据库迁移成功!');
		$this->command->info("\n请重新运行此命令!");
	}
	
	// 配置端口
	public function step_4(){
		$this->command->info($this->getTips());
		$web = $this->command->ask('WEB服务端口',env('SERVER_WEB_PORT'));
		$this->command->line($web);
		$ws = $this->command->ask('WEBSOCKET服务端口',env('SERVER_WS_PORT'));
		$this->command->line($ws);
		modifyEnv([
			'SERVER_WEB_PORT' => $web,
			'SERVER_WS_PORT' => $ws,
		]);
		$this->addStep();
		$this->command->info('配置成功!');
		$this->command->info("\nWEB服务端口:".$web."\nWEBSOCKET服务端口:".$ws);
		$this->command->info("\n请根据文档将对应端口进行反向代理!");
		$this->command->info("\n反代完成后打开网站进行最后一步安装!");
	}
	
}