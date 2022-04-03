<?php

namespace App\Plugins\MeiliSearch\src\Crontab;

use App\Model\AdminOption;
use App\Plugins\MeiliSearch\src\Data;
use Hyperf\Crontab\Annotation\Crontab;
use MeiliSearch\Client;

/**
 * @Crontab(name="PushMeiliSearch", rule="*\/60 * * * *", callback="execute", enable={Push::class, "isEnable"}, memo="PushMeiliSearch")
 */
class Push
{
	public function execute()
	{
		$data = new Data();
		$client = new Client($this->get_options("meilisearch_url",'http://127.0.0.1:7700'),$this->get_options("meilisearch_apikey",null));
		$index = $this->get_options("meilisearch_index",$this->get_options("APP_NAME","SuperForum"));
		$client->deleteIndex($index);
		$client->index($index)->addDocuments($data->get());
		$client->index($index)->updateSearchableAttributes([
			'title',
		]);
	}
	
	public function isEnable(): bool
	{
		return true;
	}
	
	public function get_options($name,$default=""){
		return $this->core_default(@AdminOption::query()->where("name",$name)->first()->value,$default);
	}
	
	public function core_default($string=null,$default=null){
		if($string){
			return $string;
		}
		return $default;
	}
}