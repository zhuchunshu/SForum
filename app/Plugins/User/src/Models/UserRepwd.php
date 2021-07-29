<?php


namespace App\Plugins\User\src\Models;


use App\Model\Model;

class UserRepwd extends Model
{
    /**
     * The table associated with the model.
     *
     * @var string
     */
    protected $table = 'user_repwd';
    /**
     * The attributes that are mass assignable.
     *
     * @var array
     */
    protected $fillable = ['id','hash','user_id','pwd'];
    /**
     * The attributes that should be cast to native types.
     *
     * @var array
     */
    protected $casts = ['id' => 'integer', 'created_at' => 'datetime', 'updated_at' => 'datetime'];
}