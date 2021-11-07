<?php

declare (strict_types=1);
namespace App\Plugins\Core\src\Models;

use App\Model\Model;
use Carbon\Carbon;

/**
 * @property int $id 
 * @property string $type 
 * @property string $_id 
 * @property string $status 
 * @property string $user_id 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class Report extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'report';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id', 'type', 'status', 'user_id','_id','created_at', 'updated_at','title','content'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
}