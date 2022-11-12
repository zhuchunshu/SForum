<?php

namespace App\Plugins\Core\src\Lib\Pay\Jobs;

use Hyperf\AsyncQueue\Job;

class OrderCloseJob extends Job
{
    public $params;

    /**
     * 任务执行失败后的重试次数，即最大执行次数为 $maxAttempts+1 次
     *
     * @var int
     */
    protected $maxAttempts = 2;

    public function __construct(array $params)
    {
        // 这里最好是普通数据，不要使用携带 IO 的对象，比如 PDO 对象
        $this->params = $params;
    }

    public function handle()
    {
        $order_id = $this->params['order_id'];
        $payServer = $this->params['payServer'];
        var_dump(pay()->close($order_id,false,$payServer));
    }
}
