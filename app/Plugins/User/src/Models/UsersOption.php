<?php

declare (strict_types=1);
namespace App\Plugins\User\src\Models;

use App\Model\Model;
use Carbon\Carbon;

/**
 * @property int $id 
 * @property string $qianming 
 * @property string $qq 
 * @property string $wx 
 * @property string $website 
 * @property string $email 
 * @property string $options 
 * @property Carbon $created_at
 * @property Carbon $updated_at
 */
class UsersOption extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'users_options';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ["id","created_at","updated_at","qianming","qq","weixin","website","email","options","credits","golds","exp","money"];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
}