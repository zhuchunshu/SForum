<?php

namespace App\Jobs;

use Alchemy\Zippy\Zippy;
use Hyperf\AsyncQueue\Annotation\AsyncQueueMessage;

class Upgrading
{
	/**
	 * @param string  $download 下载链接
	 * @param string  $path 安装包存放路径
	 * @return void
	 */
	#[AsyncQueueMessage]
	public function handle(string $download,string $path){
		// 生成更新锁
		file_put_contents(BASE_PATH."/app/CodeFec/storage/update.lock");
		// 下载文件
		file_put_contents($path,fopen($download,'r'));
		
		// 定义临时压缩包存放目录
		$tmp = BASE_PATH."/app/CodeFec/storage/update";
		
		// 初始化压缩操作类
		$zippy = Zippy::load();
		
		// 打开压缩文件
		$archiveTar  =  $zippy->open($path);
		
		// 解压
		if(!is_dir($tmp)){
			mkdir($tmp,0777);
		}
		// 解压
		$archiveTar->extract($tmp);
		
		// 获取解压后,插件文件夹的所有目录
		$allDir = allDir($tmp);
		foreach($allDir as $value){
			if(file_exists($value."/.CodeFec")){
				$dirname = file_get_contents($value."/.dirName");
				if(!$dirname){
					$this->removeFiles($tmp,$path);
				}
				FileUtil()->moveDir($value,BASE_PATH,true);
				\Swoole\Coroutine\System::exec('cd "'.BASE_PATH.'" && composer du');
				$this->removeFiles($tmp,$path);
			}
		}
	}
	
	public function removeFiles(...$values): void
	{
		foreach($values as $value){
			\Swoole\Coroutine\System::exec('rm -rf "' . $value.'"');
		}
	}
	
}