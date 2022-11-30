<?php

declare(strict_types=1);
namespace App\Plugins\Core\src\Models;

use App\Model\Model;

/**
 * @property int $id
 * @property string $original
 * @property string $cash
 * @property string $user_id
 * @property string $order_id
 * @property string $remark
 * @property \Carbon\Carbon $created_at
 * @property \Carbon\Carbon $updated_at
 */
class PayAmountRecord extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'pay_amount_record';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id','original','cash','user_id','order_id','remark','updated_at','created_at'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
}
