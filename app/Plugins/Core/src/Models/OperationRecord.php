<?php

declare (strict_types=1);
namespace App\Plugins\Core\src\Models;

use App\Model\Model;
/**
 * @property int $id 
 * @property int $user_id 
 * @property string $event 
 * @property int $event_id 
 * @property \Carbon\Carbon $created_at 
 * @property \Carbon\Carbon $updated_at 
 */
class OperationRecord extends Model
{
    protected ?string $dateFormat = 'U';
    // 使用 Unix 时间戳格式
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected ?string $table = 'operation_record';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected array $fillable = ['id', 'user_id', 'event', 'event_id', 'created_at', 'updated_at'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected array $casts = ['id' => 'integer', 'user_id' => 'integer', 'event_id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
}