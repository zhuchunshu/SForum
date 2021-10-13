<?php

declare (strict_types=1);
namespace App\Plugins\User\src\Models;

use App\Model\Model;
use Carbon\Carbon;

/**
 * @property int $id 
 * @property string $user_id 
 * @property int $email 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class UsersNoticed extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'users_noticed';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['user_id','email','created_at','updated_at'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'email' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
}