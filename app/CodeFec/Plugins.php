<?php
namespace App\CodeFec;

use App\Model\AdminPlugin;
use Noodlehaus\Config;

class Plugins {

    public static function GetAll(): array
    {
        $arr = getPath(plugin_path());
        $plugin_arr = [];
        foreach ($arr as $value) {
            if(file_exists(plugin_path($value."/".$value.".php"))){
				// 插件目录名
                $plugin_arr[$value]['dir']=$value;
				// 插件路径
                $plugin_arr[$value]['path']=plugin_path($value);
				// 插件类
                $plugin_arr[$value]['class']="\App\Plugins\\".$value."\\".$value;
				// 插件信息
                $plugin_arr[$value]['data']=get_plugins_doc($plugin_arr[$value]['class']);
				if(file_exists(plugin_path($value."/".$value.".json"))){
					$plugin_arr[$value]['data'] = array_merge($plugin_arr[$value]['data'],Config::load(plugin_path($value."/".$value.".json"))->all());
				}
				
                $plugin_arr[$value]['file']=plugin_path($value."/".$value.".php");
            }
        }
        return $plugin_arr;
    }
	
	/**
	 * @param string $dirName 插件目录名
	 * @return ?string
	 */
	public function getLogo(string $dirName): ?string
	{
		// 插件logo
		$file = plugin_path($dirName."/".$dirName.".png");
		if(file_exists($file)){
			$image_info = getimagesize($file);
			$image_data = fread(fopen($file, 'rb'), filesize($file));
			return 'data:' . $image_info['mime'] . ';base64,' . chunk_split(base64_encode($image_data));
		}
		return null;
	}

    // 获取已启用的插件列表
    public function getEnPlugins(){
        $plugins = ['Core','Mail','User','Topic','Comment','Search'];
        if(!file_exists(BASE_PATH."/app/CodeFec/storage/install.lock")){
            return $plugins;
        }
        if(!cache()->has("plugins.en")){
            $array = AdminPlugin::query()->where("status",1)->get();
            $result = [];
            foreach ($array as $value) {
                $result[]=$value->name;
            }
            $result = array_merge($plugins,$result);
            cache()->set("plugins.en",array_unique($result));
            return array_values(array_unique($result));
        }
        return array_values(array_unique(cache()->get("plugins.en")));

    }

    public function composerInstall(): void
    {
        $list = self::GetAll();
        foreach($list as $data){
            if(file_exists(plugin_path($data['dir']."/composer.json"))){
                 \Swoole\Coroutine\System::exec("cd ".plugin_path($data['dir'])." && yes yes | composer update");
            }
        }
    }

}