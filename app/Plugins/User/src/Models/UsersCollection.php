<?php

declare (strict_types=1);
namespace App\Plugins\User\src\Models;

use App\Model\Model;
use Carbon\Carbon;

/**
 * @property int $id 
 * @property string $title 
 * @property string $content 
 * @property string $action 
 * @property string $user_id 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class UsersCollection extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'users_collections';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id','user_id','created_at','updated_at','type','type_id'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
}