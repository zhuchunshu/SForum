<?php

declare (strict_types=1);
namespace App\Plugins\User\src\Models;

use Carbon\Carbon;

/**
 * @property int $id 
 * @property string $name 
 * @property string $color 
 * @property string $icon 
 * @property string $quanxian 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class UserClass extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'user_class';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id','name','quanxian','icon','color'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
}