<?php

namespace App\CodeFec\Admin;

use Hyperf\Utils\Str;
use SleekDB\Query;
use SleekDB\Store;

/**
 * 日志服务
 */
class LogServer
{
    private string $store;

    private string $name;

    private string $log;

    private string|array|object $data;

    private array $db_config = [
        "auto_cache" => true,
        "cache_lifetime" => null,
        "timeout" => 120, // deprecated! Set it to false!
        "primary_key" => "_id",
        "search" => [
            "min_length" => 2,
            "mode" => "or",
            "score_key" => "scoreKey",
            "algorithm" => Query::SEARCH_ALGORITHM["hits"]
        ],
        "folder_permissions" => 0777
    ];

    /**
     * nosql
     * @return Store
     * @throws \SleekDB\Exceptions\IOException
     * @throws \SleekDB\Exceptions\InvalidArgumentException
     * @throws \SleekDB\Exceptions\InvalidConfigurationException
     */
    public function db(): Store
    {
        if(!is_dir(BASE_PATH."/runtime/logs/admin_logger_database")){
            mkdir(BASE_PATH."/runtime/logs/admin_logger_database");
        }
        $databasePath = BASE_PATH."/runtime/logs/admin_logger_database/".date('YmW');
        $userStore = @new Store('logger', $databasePath,$this->db_config);
        return $userStore;
    }

    /**
     * 插入日志数据
     * @param $name
     * @param $log
     * @param $data
     * @return void
     * @throws \SleekDB\Exceptions\IOException
     * @throws \SleekDB\Exceptions\IdNotAllowedException
     * @throws \SleekDB\Exceptions\InvalidArgumentException
     * @throws \SleekDB\Exceptions\JsonException
     */
    public function insert($store,$name, $log, $data){
        $this->store = $store;
        $this->name = $name;
        $this->log = $log;
        $this->data = $data;
        $this->db()->insert([
            'store' => $this->store,
            'name' => $this->name,
            'log' => $this->log,
            'data' => $this->data,
            '_token' => Str::random(),
            'created_at' => date("Y-m-d H:i:s")
        ]);
    }

    /**
     * 获取所有数据
     * @return array
     * @throws \SleekDB\Exceptions\IOException
     * @throws \SleekDB\Exceptions\InvalidArgumentException
     */
    public function get(): array
    {
        return $this->db()->createQueryBuilder()->orderBy(['created_at' => 'desc','id'=>'desc'])->getQuery()->fetch();
    }

}