#include "seal/seal.h"
#include <iostream>

using namespace std;
using namespace seal;

int main() {
    // Set up the encryption parameters
    EncryptionParameters parms(scheme_type::bfv);
    size_t poly_modulus_degree = 4096;
    parms.set_poly_modulus_degree(poly_modulus_degree);
    parms.set_coeff_modulus(CoeffModulus::BFVDefault(poly_modulus_degree));
    parms.set_plain_modulus(40961);
    SEALContext context(parms); 


    // Key generation
    KeyGenerator keygen(context);
    PublicKey public_key;
    keygen.create_public_key(public_key);
    SecretKey secret_key = keygen.secret_key();

    // Encryption and decryption setup
    Encryptor encryptor(context, public_key);
    Decryptor decryptor(context, secret_key);

    // Evaluator for performing operations
    Evaluator evaluator(context);

    // Encoder setup
    BatchEncoder encoder(context);
    vector<uint64_t> input1(poly_modulus_degree / 2, 10);
    vector<uint64_t> input2(poly_modulus_degree / 2, 20);
    Plaintext plain1, plain2;
    encoder.encode(input1, plain1);
    encoder.encode(input2, plain2);

    // Encrypt two integers
    Ciphertext encrypted1, encrypted2;
    encryptor.encrypt(plain1, encrypted1);
    encryptor.encrypt(plain2, encrypted2);

    // Perform homomorphic addition
    Ciphertext encrypted_result;
    evaluator.add(encrypted1, encrypted2, encrypted_result);

    // Decrypt the result
    Plaintext decrypted_result;
    decryptor.decrypt(encrypted_result, decrypted_result);
    vector<uint64_t> result;
    encoder.decode(decrypted_result, result);

    // Output the result
    cout << "Result of homomorphic addition: " << result[0] << endl;

    return 0;
}
