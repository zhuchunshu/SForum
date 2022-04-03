<?php


declare(strict_types=1);

namespace App\Plugins\MeiliSearch\src\Command;

use App\Model\AdminOption;
use App\Plugins\MeiliSearch\src\Data;
use Hyperf\Command\Command as HyperfCommand;
use Hyperf\Command\Annotation\Command;
use MeiliSearch\Client;
use Psr\Container\ContainerInterface;

/**
 * @Command
 */
#[Command]
class PushMeiliSearch extends HyperfCommand
{
	/**
	 * @var ContainerInterface
	 */
	protected ContainerInterface $container;
	
	public function __construct(ContainerInterface $container)
	{
		$this->container = $container;
		
		parent::__construct('MeiliSearch:push');
	}
	
	public function configure()
	{
		parent::configure();
		$this->setDescription('MeiliSearch Push');
	}
	
	public function handle()
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
