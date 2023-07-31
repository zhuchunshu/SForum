<?php

namespace App\Plugins\Core\src\Lib\Pay\Jobs\Service;

use App\Plugins\Core\src\Lib\Pay\Jobs\OrderCloseJob;
use Hyperf\AsyncQueue\Driver\DriverFactory;
use Hyperf\AsyncQueue\Driver\DriverInterface;
class OrderCloseJobService
{
    /**
     * @var DriverInterface
     */
    protected DriverInterface $driver;
    public function __construct(DriverFactory $driverFactory)
    {
        $this->driver = $driverFactory->get('default');
    }
    /**
     * 生产消息.
     * @param array $params 传递参数
     * @param int $delay 延时时间 单位秒
     * @return bool
     */
    public function push(array $params, int $delay = 300) : bool
    {
        // 这里的 `ExampleJob` 会被序列化存到 Redis 中，所以内部变量最好只传入普通数据
        // 同理，如果内部使用了注解 @Value 会把对应对象一起序列化，导致消息体变大。
        // 所以这里也不推荐使用 `make` 方法来创建 `Job` 对象。
        return $this->driver->push(new OrderCloseJob($params), $delay);
    }
}