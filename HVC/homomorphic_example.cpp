#include "seal/seal.h"
#include <iostream>
#include <vector>

using namespace std;
using namespace seal;

extern "C" {
    void performHomomorphicAddition(int input1_value, int input2_value) {
        EncryptionParameters parms(scheme_type::bfv);
        size_t poly_modulus_degree = 4096;
        parms.set_poly_modulus_degree(poly_modulus_degree);
        parms.set_coeff_modulus(CoeffModulus::BFVDefault(poly_modulus_degree));
        parms.set_plain_modulus(40961);
        SEALContext context(parms);

        KeyGenerator keygen(context);
        PublicKey public_key;
        keygen.create_public_key(public_key);
        SecretKey secret_key = keygen.secret_key();

        Encryptor encryptor(context, public_key);
        Decryptor decryptor(context, secret_key);
        Evaluator evaluator(context);
        BatchEncoder encoder(context);

        vector<uint64_t> input1(poly_modulus_degree / 2, input1_value);
        vector<uint64_t> input2(poly_modulus_degree / 2, input2_value);
        Plaintext plain1, plain2;
        encoder.encode(input1, plain1);
        encoder.encode(input2, plain2);

        Ciphertext encrypted1, encrypted2;
        encryptor.encrypt(plain1, encrypted1);
        encryptor.encrypt(plain2, encrypted2);

        Ciphertext encrypted_result;
        evaluator.add(encrypted1, encrypted2, encrypted_result);

        Plaintext decrypted_result;
        decryptor.decrypt(encrypted_result, decrypted_result);
        vector<uint64_t> result;
        encoder.decode(decrypted_result, result);

        cout << "Result of homomorphic addition: " << result[0] << endl;
    }
}
