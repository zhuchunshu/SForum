<?php

declare (strict_types=1);
namespace App\Plugins\Core\src\Models;

use App\Model\Model;
use App\Plugins\User\src\Models\User;
/**
 * @property int $id 订单号
 * @property string $title 订单标题
 * @property string $status 订单状态
 * @property string $user_id 订单发起者id
 * @property string $amount 订单金额
 * @property \Carbon\Carbon $created_at 订单创建时间
 * @property \Carbon\Carbon $updated_at 订单最后更新时间
 * @property string $trade_no 订单交易单号
 * @property string $payer_total 用户支付金额
 * @property string $payment_method 支付方式
 * @property string $notify_result 回调通知结果
 * @property string $amount_total 总金额
 */
class PayOrder extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected ?string $table = 'pay_order';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected array $fillable = ['id', 'title', 'description', 'status', 'payment_method', 'user_id', 'amount', 'trade_no', 'payer_total', 'notify_result', 'created_at', 'updated_at', 'amount_total'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected array $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
    public function user()
    {
        return $this->belongsTo(User::class, "user_id", "id");
    }
}